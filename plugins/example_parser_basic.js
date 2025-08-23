// 这是一个最简示例解析器插件, 用于展示插件所需实现的基本功能
// 此插件将会模拟处理 YouTube 的视频链接

/**
 * 插件元数据
 * 版本号是 saveany-bot 本体支持的插件规范版本号, 必须提供
 */
const metadata = {
    name: "Example Parser", // 插件名称
    version: "1.0.0", // 插件版本号
    description: "A parser for example links", // 插件描述
    author: "Krau", // 插件作者
}

// 你可以使用 console.log 来在终端中使用 go 的 logger 打印信息
console.log("Parser loaded", "name", metadata.name);

/**
 * canHandle 函数用于判断当前解析器能否解析给定的 URL
 */
const canHandle = function (url) {
    // 这里我们简单地检查 URL 是否包含 "youtube.com/watch?v"
    return url.includes("youtube.com/watch?v");
}

/**
 * 解析 url 并返回一个 Item 对象, 类型定义在 pkg/parser.go 中
 */
const parse = function (url) {
    var result = {
        // 元信息
        site: "YouTube",
        url: url,
        title: "测试 YouTube 视频",
        author: "某视频作者",
        description: "这是一个测试视频",
        tags: ["test", "youtube"],
        // 资源(可下载的文件)列表
        resources: [
            {
                url: "https://example.com/video1.mp4", // 文件直链
                filename: "somevideo.mp4", // 文件名
                mime_type: "video/mp4", // 文件 MIME 类型, 可选
                extension: "mp4", // 文件扩展名, 可选
                size: 100 * 1024 * 1024, // 文件大小, 单位为字节, 未知可以设置为 0
                hash: {}, // 文件哈希, 可选, 格式为 {"md5": "xxx", "sha256": "xxx"} 等
                headers: {}, // 下载文件时所需的 HTTP 头部, 可选, 例如 {"User-Agent": "Mozilla/5.0"}
                extra: {} // 额外信息, 可选, 可以包含任何自定义数据
            },
            {
                url: "https://example.com/picture1.png",
                filename: "picture1.png",
                mime_type: "image/png",
                extension: "png",
                size: 1 * 1024 * 1024,
                hash: {},
                headers: {},
                extra: {}
            }
        ],
        extra: {}
    };
    return result;
}

// 最后需要调用 registerParser 来注册这个解析器
registerParser({
    metadata,
    canHandle,
    parse
});

// 更进一步的插件编写信息, 请查看 plugins/example_parser_danbooru.js