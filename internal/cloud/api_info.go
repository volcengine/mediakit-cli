package cloud

type APIInfo struct {
	Method string
	Path   string
}

var apiInfoRegistry = map[string]APIInfo{
	"erase-video-subtitle-pro": {Method: "POST", Path: "/api/v1/tools/erase-video-subtitle-pro"},
	"image-to-video":           {Method: "POST", Path: "/api/v1/tools/image-to-video"},
	"extract-audio":            {Method: "POST", Path: "/api/v1/tools/extract-audio"},
	"add-image-to-video":       {Method: "POST", Path: "/api/v1/tools/add-image-to-video"},
	"add-subtitle-to-video":    {Method: "POST", Path: "/api/v1/tools/add-subtitle-to-video"},
	"mux-audio-video":          {Method: "POST", Path: "/api/v1/tools/mux-audio-video"},
	"concat-video":             {Method: "POST", Path: "/api/v1/tools/concat-video"},
	"flip-video":               {Method: "POST", Path: "/api/v1/tools/flip-video"},
	"trim-video":               {Method: "POST", Path: "/api/v1/tools/trim-video"},
	"adjust-video-speed":       {Method: "POST", Path: "/api/v1/tools/adjust-video-speed"},
	"concat-audio":             {Method: "POST", Path: "/api/v1/tools/concat-audio"},
	"trim-audio":               {Method: "POST", Path: "/api/v1/tools/trim-audio"},
	"enhance-video":            {Method: "POST", Path: "/api/v1/tools/enhance-video"},
	"query-task":               {Method: "GET", Path: "/api/v1/tasks/{task_id}"},
}
