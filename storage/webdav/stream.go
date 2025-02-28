package webdav

// TODO: gowebdav's WriteStream impl cause high memory usage, need to implement our own WriteStream
// type WebdavWriter struct {
// 	pipeWriter *io.PipeWriter
// 	done       chan error
// 	path       string
// }

// func (w *WebdavWriter) Write(p []byte) (n int, err error) {
// 	return w.pipeWriter.Write(p)
// }

// func (w *WebdavWriter) Close() error {
// 	if err := w.pipeWriter.Close(); err != nil {
// 		return err
// 	}
// 	if err := <-w.done; err != nil {
// 		return fmt.Errorf("upload failed: %w", err)
// 	}

// 	return nil
// }

// func (w *Webdav) NewUploadStream(ctx context.Context, storagePath string) (io.WriteCloser, error) {
// 	if err := w.client.MkdirAll(path.Dir(storagePath), os.ModePerm); err != nil {
// 		logger.L.Errorf("Failed to create directory %s: %v", path.Dir(storagePath), err)
// 		return nil, ErrFailedToCreateDirectory
// 	}
// 	pipeReader, pipeWriter := io.Pipe()
// 	done := make(chan error, 1)
// 	go func() {
// 		defer func() {
// 			if err := recover(); err != nil {
// 				done <- fmt.Errorf("panic during upload: %v", err)
// 			}
// 		}()

// 		err := w.client.WriteStream(storagePath, pipeReader, os.ModePerm)

// 		pipeReader.Close()
// 		done <- err
// 	}()

// 	return &WebdavWriter{
// 		pipeWriter: pipeWriter,
// 		done:       done,
// 		path:       storagePath,
// 	}, nil
// }
