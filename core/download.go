package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/celestix/gotgproto/ext"
	"github.com/celestix/telegraph-go/v2"
	"github.com/duke-git/lancet/v2/fileutil"
	"github.com/gotd/td/telegram/message/entity"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/bot"
	"github.com/krau/SaveAny-Bot/common"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/krau/SaveAny-Bot/types"
	"golang.org/x/sync/errgroup"
)

func processPendingTask(task *types.Task) error {
	common.Log.Debugf("Start processing task: %s", task.String())
	if task.FileName() == "" {
		task.File.FileName = fmt.Sprintf("%d_%d_%s", task.FileChatID, task.FileMessageID, task.File.Hash())
	}

	taskStorage, storagePath, err := getStorageAndPathForTask(task)
	if err != nil {
		return err
	}
	if taskStorage == nil {
		return fmt.Errorf("not found storage: %s", task.StorageName)
	}
	task.StoragePath = storagePath

	ctx, ok := task.Ctx.(*ext.Context)
	if !ok {
		return fmt.Errorf("context is not *ext.Context: %T", task.Ctx)
	}

	cancelCtx, cancel := context.WithCancel(ctx)
	task.Cancel = cancel

	if task.IsTelegraph {
		return processTelegraph(ctx, cancelCtx, task, taskStorage)
	}

	if task.File.FileSize == 0 {
		return processPhoto(task, taskStorage)
	}

	downloadBuilder := Downloader.Download(bot.Client.API(), task.File.Location).WithThreads(getTaskThreads(task.File.FileSize))

	notsupportStreamStorage, notsupportStream := taskStorage.(storage.StorageNotSupportStream)
	cancelMarkUp := getCancelTaskMarkup(task)
	if config.Cfg.Stream {
		if !notsupportStream {
			text, entities := buildProgressMessageEntity(task, 0, task.StartTime, 0)
			if task.ReplyMessageID != 0 {
				ctx.EditMessage(task.ReplyChatID, &tg.MessagesEditMessageRequest{
					Message:     text,
					Entities:    entities,
					ID:          task.ReplyMessageID,
					ReplyMarkup: cancelMarkUp,
				})
			}

			pr, pw := io.Pipe()
			defer pr.Close()

			task.StartTime = time.Now()
			progressCallback := buildProgressCallback(ctx, task, getProgressUpdateCount(task.File.FileSize))

			progressStream := NewProgressStream(pw, task.File.FileSize, progressCallback)

			eg, uploadCtx := errgroup.WithContext(cancelCtx)

			eg.Go(func() error {
				return taskStorage.Save(uploadCtx, pr, task.StoragePath)
			})
			eg.Go(func() error {
				_, err := downloadBuilder.Stream(uploadCtx, progressStream)
				if closeErr := pw.CloseWithError(err); closeErr != nil {
					common.Log.Errorf("Failed to close pipe writer: %v", closeErr)
				}
				return err
			})
			if err := eg.Wait(); err != nil {
				return err
			}

			return nil
		}
		common.Log.Warnf("存储 %s 不支持流式传输: %s", task.StorageName, notsupportStreamStorage.NotSupportStream())

		if task.ReplyMessageID != 0 {
			ctx.EditMessage(task.ReplyChatID, &tg.MessagesEditMessageRequest{
				Message:     fmt.Sprintf("存储 %s 不支持流式传输: %s\n正在使用普通下载...", task.StorageName, notsupportStreamStorage.NotSupportStream()),
				ID:          task.ReplyMessageID,
				ReplyMarkup: cancelMarkUp,
			})
		}
	}

	cacheDestPath := filepath.Join(config.Cfg.Temp.BasePath, task.FileName())
	cacheDestPath, err = filepath.Abs(cacheDestPath)
	if err != nil {
		return fmt.Errorf("处理路径失败: %w", err)
	}
	if err := fileutil.CreateDir(filepath.Dir(cacheDestPath)); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	text, entities := buildProgressMessageEntity(task, 0, task.StartTime, 0)
	if task.ReplyMessageID != 0 {
		ctx.EditMessage(task.ReplyChatID, &tg.MessagesEditMessageRequest{
			Message:     text,
			Entities:    entities,
			ID:          task.ReplyMessageID,
			ReplyMarkup: cancelMarkUp,
		})
	}

	progressCallback := buildProgressCallback(ctx, task, getProgressUpdateCount(task.File.FileSize))
	dest, err := NewTaskLocalFile(cacheDestPath, task.File.FileSize, progressCallback)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer dest.Close()
	task.StartTime = time.Now()
	_, err = downloadBuilder.Parallel(cancelCtx, dest)
	if err != nil {
		return fmt.Errorf("下载文件失败: %w", err)
	}
	defer cleanCacheFile(cacheDestPath)

	fixTaskFileExt(task, cacheDestPath)

	common.Log.Infof("Downloaded file: %s", cacheDestPath)
	if task.ReplyMessageID != 0 {
		ctx.EditMessage(task.ReplyChatID, &tg.MessagesEditMessageRequest{
			Message: fmt.Sprintf("下载完成: %s\n正在转存文件...", task.FileName()),
			ID:      task.ReplyMessageID,
		})
	}
	return saveFileWithRetry(cancelCtx, task.StoragePath, taskStorage, cacheDestPath)
}

func processTelegraph(extCtx *ext.Context, cancelCtx context.Context, task *types.Task, taskStorage storage.Storage) error {
	if bot.TelegraphClient == nil {
		return fmt.Errorf("telegraph client is not initialized")
	}
	tgphUrl := task.TelegraphURL
	tgphPath := strings.Split(tgphUrl, "/")[len(strings.Split(tgphUrl, "/"))-1]
	if tgphUrl == "" || tgphPath == "" {
		return fmt.Errorf("invalid telegraph url")
	}
	entityBuilder := entity.Builder{}
	text := fmt.Sprintf("正在下载 Telegraph \n文件夹: %s\n保存路径: %s",
		task.FileName(),
		fmt.Sprintf("[%s]:%s", task.StorageName, task.StoragePath),
	)
	var entities []tg.MessageEntityClass
	if err := styling.Perform(&entityBuilder,
		styling.Plain("正在下载 Telegraph \n文件夹: "),
		styling.Code(task.FileName()),
		styling.Plain("\n保存路径: "),
		styling.Code(fmt.Sprintf("[%s]:%s", task.StorageName, task.StoragePath)),
	); err != nil {
		common.Log.Errorf("Failed to build entities: %s", err)
	}

	if task.ReplyMessageID != 0 {
		extCtx.EditMessage(task.ReplyChatID, &tg.MessagesEditMessageRequest{
			Message:     text,
			Entities:    entities,
			ID:          task.ReplyMessageID,
			ReplyMarkup: getCancelTaskMarkup(task),
		})
	}

	resultCh := make(chan error)
	go func() {
		page, err := bot.TelegraphClient.GetPage(tgphPath, true)
		if err != nil {
			resultCh <- fmt.Errorf("获取 telegraph 页面失败: %w", err)
			return
		}
		imgs := make([]string, 0)
		for _, element := range page.Content {
			var node telegraph.NodeElement
			data, err := json.Marshal(element)
			if err != nil {
				common.Log.Errorf("Failed to marshal element: %s", err)
				continue
			}
			err = json.Unmarshal(data, &node)
			if err != nil {
				common.Log.Errorf("Failed to unmarshal element: %s", err)
				continue
			}

			if len(node.Children) != 0 {
				for _, child := range node.Children {
					imgs = append(imgs, getNodeImages(child)...)
				}
			}

			if node.Tag == "img" {
				if src, ok := node.Attrs["src"]; ok {
					imgs = append(imgs, src)
				}
			}

		}
		if len(imgs) == 0 {
			resultCh <- fmt.Errorf("没有找到图片")
			return
		}
		hc := bot.TelegraphClient.HttpClient
		eg, ectx := errgroup.WithContext(cancelCtx)
		eg.SetLimit(config.Cfg.Workers) // TODO: use a new config field for this
		for i, img := range imgs {
			if strings.HasPrefix(img, "/file/") {
				img = "https://telegra.ph" + img
			}
			eg.Go(func() error {
				var lastErr error
				for attempt := range config.Cfg.Retry {
					if attempt > 0 {
						retryDelay := time.Duration(attempt*attempt) * time.Second
						select {
						case <-ectx.Done():
							return ectx.Err()
						case <-time.After(retryDelay):
						}
						common.Log.Debugf("Retrying to download image %s (attempt %d)", img, attempt+1)
					}
					req, err := http.NewRequestWithContext(ectx, http.MethodGet, img, nil)
					if err != nil {
						lastErr = fmt.Errorf("创建请求失败: %w", err)
						continue
					}
					resp, err := hc.Do(req)
					if err != nil {
						lastErr = fmt.Errorf("发送请求失败: %w", err)
						continue
					}
					defer resp.Body.Close()
					if resp.StatusCode != http.StatusOK {
						lastErr = fmt.Errorf("请求图片失败: %s", resp.Status)
						continue
					}
					targetPath := path.Join(task.StoragePath, fmt.Sprintf("%d%s", i+1, path.Ext(img)))
					err = taskStorage.Save(ectx, resp.Body, targetPath)
					if err != nil {
						lastErr = fmt.Errorf("保存图片失败: %w", err)
						continue
					}
					common.Log.Infof("Saved image: %s", targetPath)
					return nil
				}
				return lastErr
			})
		}
		if err := eg.Wait(); err != nil {
			resultCh <- err
			return
		}
		resultCh <- nil
	}()
	select {
	case err := <-resultCh:
		return err
	case <-cancelCtx.Done():
		return cancelCtx.Err()
	}
}
