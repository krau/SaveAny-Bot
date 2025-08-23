// Danbooru post parser for SaveAnyBot
// request https://danbooru.donmai.us/posts/{id}.json and parse the response

const metadata = {
    name: "Danbooru Post Parser",
    version: "1.0.0",
    description: "Parse Danbooru post links via official JSON API",
    author: "Krau",
};

// some utils
const danbooruSourceURLRegexp = /danbooru\.donmai\.us\/(posts|post\/show)\/(\d+)/;
function getPostID(url) {
    const m = url.match(danbooruSourceURLRegexp);
    return m ? m[2] : "";
}
function normalizePostURL(id) {
    return `https://danbooru.donmai.us/posts/${id}`;
}
function apiURLFor(id) {
    return `https://danbooru.donmai.us/posts/${id}.json`;
}


function basenameFromURL(u) {
    try {
        const q = u.split("?")[0];
        const parts = q.split("/");
        const name = parts[parts.length - 1] || "";
        return name || "file";
    } catch (_) {
        return "file";
    }
}
function extFromFilename(name) {
    const idx = name.lastIndexOf(".");
    if (idx < 0) return "";
    return name.slice(idx + 1).toLowerCase();
}
function mimeFromExt(ext) {
    switch (ext) {
        case "jpg":
        case "jpeg":
            return "image/jpeg";
        case "png":
            return "image/png";
        case "gif":
            return "image/gif";
        default:
            return "";
    }
}

// implement canHandle and parse
const canHandle = function (url) {
    return danbooruSourceURLRegexp.test(url);
};

const parse = function (sourceURL) {
    const id = getPostID(sourceURL);
    if (!id) {
        throw new Error("invalid danbooru post url");
    }

    const normURL = normalizePostURL(id);

    const apiURL = apiURLFor(id);
    console.log("Danbooru requesting", "url", apiURL);
    // You can use ghttp.getJSON to fetch and parse JSON in one step.
    // While the ghttp.get can be used to fetch raw response.
    const data = ghttp.getJSON(apiURL);

    if (data && data.error) {
        throw new Error(data.message || "danbooru returned error");
    }

    const fileURL = data.file_url || "";
    const largeURL = data.large_file_url || "";
    const width = data.image_width || 0;
    const height = data.image_height || 0;

    if (!fileURL && !largeURL) {
        throw new Error("danbooru response has no file_url / large_file_url");
    }

    const resources = [];
    if (fileURL) {
        const name = basenameFromURL(fileURL);
        const ext = extFromFilename(name);
        resources.push({
            url: fileURL,
            filename: name,
            mime_type: mimeFromExt(ext),
            extension: ext,
            size: 0,
            hash: {},
            headers: {},
            extra: { width, height, kind: "original" },
        });
    }
    if (largeURL && largeURL !== fileURL) {
        const name = basenameFromURL(largeURL);
        const ext = extFromFilename(name);
        resources.push({
            url: largeURL,
            filename: name,
            mime_type: mimeFromExt(ext),
            extension: ext,
            size: 0,
            hash: {},
            headers: {},
            extra: { width, height, kind: "large" },
        });
    }

    const tags = (data.tag_string ? String(data.tag_string) : "")
        .split(" ")
        .filter(Boolean);

    const item = {
        site: "Danbooru",
        url: normURL,
        title: `Danbooru/${data.id || id}`,
        author: "Danbooru",
        description: "",
        tags: tags,
        resources: resources,
        extra: {},
    };

    return item;
};

registerParser({
    metadata,
    canHandle,
    parse,
});
