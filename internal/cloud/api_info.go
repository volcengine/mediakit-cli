package cloud

type APIInfo struct {
	Method string
	Path   string
}

var apiInfoRegistry = map[string]APIInfo{
	"erase-video-subtitle-pro":       {Method: "POST", Path: "/api/v1/tools/erase-video-subtitle-pro"},
	"image-to-video":                 {Method: "POST", Path: "/api/v1/tools/image-to-video"},
	"extract-audio":                  {Method: "POST", Path: "/api/v1/tools/extract-audio"},
	"add-image-to-video":             {Method: "POST", Path: "/api/v1/tools/add-image-to-video"},
	"add-subtitle-to-video":          {Method: "POST", Path: "/api/v1/tools/add-subtitle-to-video"},
	"mux-audio-video":                {Method: "POST", Path: "/api/v1/tools/mux-audio-video"},
	"concat-video":                   {Method: "POST", Path: "/api/v1/tools/concat-video"},
	"flip-video":                     {Method: "POST", Path: "/api/v1/tools/flip-video"},
	"trim-video":                     {Method: "POST", Path: "/api/v1/tools/trim-video"},
	"adjust-video-speed":             {Method: "POST", Path: "/api/v1/tools/adjust-video-speed"},
	"concat-audio":                   {Method: "POST", Path: "/api/v1/tools/concat-audio"},
	"trim-audio":                     {Method: "POST", Path: "/api/v1/tools/trim-audio"},
	"enhance-video":                  {Method: "POST", Path: "/api/v1/tools/enhance-video"},
	"erase-video-subtitle":           {Method: "POST", Path: "/api/v1/tools/erase-video-subtitle"},
	"video-ocr":                      {Method: "POST", Path: "/api/v1/tools/video-ocr"},
	"asr-subtitles":                  {Method: "POST", Path: "/api/v1/tools/asr-subtitles"},
	"separate-voice":                 {Method: "POST", Path: "/api/v1/tools/separate-voice"},
	"enhance-video-generative":       {Method: "POST", Path: "/api/v1/tools/enhance-video-generative"},
	"generate-highlights-minigame":   {Method: "POST", Path: "/api/v1/tools/generate-highlights-minigame"},
	"generate-highlights-microdrama": {Method: "POST", Path: "/api/v1/tools/generate-highlights-microdrama"},
	"segment-scenes":                 {Method: "POST", Path: "/api/v1/tools/segment-scenes"},
	"analyze-video-storyline":        {Method: "POST", Path: "/api/v1/tools/analyze-video-storyline"},
	"analyze-video-highlights":       {Method: "POST", Path: "/api/v1/tools/analyze-video-highlights"},
	"matte-portrait-video":           {Method: "POST", Path: "/api/v1/tools/matte-portrait-video"},
	"matte-greenscreen-video":        {Method: "POST", Path: "/api/v1/tools/matte-greenscreen-video"},
	"probe-video-metadata":           {Method: "POST", Path: "/api/v1/tools/probe-video-metadata"},
	"fade-video-audio":               {Method: "POST", Path: "/api/v1/tools/fade-video-audio"},
	"apply-video-filter":             {Method: "POST", Path: "/api/v1/tools/apply-video-filter"},
	"adjust-video-volume":            {Method: "POST", Path: "/api/v1/tools/adjust-video-volume"},
	"fade-audio":                     {Method: "POST", Path: "/api/v1/tools/fade-audio"},
	"mix-audio":                      {Method: "POST", Path: "/api/v1/tools/mix-audio"},
	"adjust-audio-speed":             {Method: "POST", Path: "/api/v1/tools/adjust-audio-speed"},
	"probe-audio-metadata":           {Method: "POST", Path: "/api/v1/tools/probe-audio-metadata"},
	"image-ocr":                      {Method: "POST", Path: "/api/v1/tools-sync/image-ocr"},
	"erase-image":                    {Method: "POST", Path: "/api/v1/tools-sync/erase-image"},
	"remove-image-background":        {Method: "POST", Path: "/api/v1/tools-sync/remove-image-background"},
	"enhance-image":                  {Method: "POST", Path: "/api/v1/tools-sync/enhance-image"},
	"evaluate-image-quality":         {Method: "POST", Path: "/api/v1/tools-sync/evaluate-image-quality"},
	"query-task":                     {Method: "GET", Path: "/api/v1/tasks/{task_id}"},
}
