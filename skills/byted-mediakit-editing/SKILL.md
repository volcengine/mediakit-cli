---
name: byted-mediakit-editing
version: "0.2.2"
license: "MIT"
description: "面向音频、视频或图片素材组成成片的编辑制作目标，适用于时间线裁剪与拼接、速度和音量调整、视频滤镜、运镜特效、转场、画面裁切旋转翻转、画面叠加、字幕压制、动图截取、淡入淡出、音视频提取与合流、音频混合、文字滚屏成片、图转视频以及多画面空间组合等操作。若用户要给视频添加滤镜效果，或对象和目标族已明确是对现有素材做剪辑、合成、叠加、混合或成片编排，但具体做法不确定，可先加载本 Skill 探索。"
permissions:
  - shell
metadata:
  requires:
    bins: ["mediakit-cli"]
  cliHelp: "mediakit-cli editing --help"
  product: mediakit-cli/skills
  domain: editing
  capability_count: 24
---
# editing MediaKit Skill

## 使用规则

1. 先读取 `../byted-mediakit-shared/SKILL.md`，执行统一前置检查；该 Skill 缺失时停止并提示安装。
2. 只从下表选择 `editing` 域工具；相似能力按各工具“能力描述”和参数边界区分。
3. 执行前按需读取对应 reference；参数与结果说明来自同一份已审核文案，完整机器合同以当前 CLI `--schema` 为准。
4. 缺少必填参数、鉴权环境变量或真实输入资源时，向用户索取；通用可选字段只能透传用户明确提供的值，其他可选字段可由明确意图准确确定，但不得伪造。
5. 执行时设置 `MEDIAKIT_SURFACE=skill`，指定调用来源是 Skill。
6. 执行时把 `MEDIAKIT_RUNTIME` 设置为当前 Agent 宿主，避免 CLI 无法可靠识别父级 Agent。

## 音视频元数据前置探测

editing 使用现成素材时，若目标工具的限制或参数选择依赖输入时长、分辨率、文件大小、封装、编码、码率、帧率、采样率或声道，而用户没有提供可信元信息，必须按实际输入类型先探测：音频使用 audio 域的 `probe-audio-metadata`，视频使用 video 域的 `probe-video-metadata`；同时含音视频输入时分别探测。将结果与最终工具 reference 的限制逐项比较；确认超限或不兼容时停止并要求更换或调整输入，不得盲目调用。探测不能替代目标工具的专属约束，也不保证后端一定处理成功。

## 字幕添加能力边界

`add-subtitle-to-video` 只用于把字幕文件或字幕文本按工具支持的预设位置和样式压制到视频中。它不能擦除字幕或水印，不能添加水印，不支持任意 X/Y 坐标；不得把字幕文本冒充为可自由定位的文字水印层。水印诉求只能选择当前产品明确公开的水印或画面叠加工具；没有匹配能力时应明确说明不支持，不能用字幕工具代替。

## 澄清与跨域路由

若只给出媒体类型而未说明要剪、合、叠、调、加滤镜还是分析，应先澄清。纯图像增强、抠图、OCR、图片水印或图片尺寸优化应路由到 image；视频画质增强、内容理解、从视频提取字幕、语音转字幕、字幕擦除、暗水印、人像或绿幕抠像等视频智能处理应路由到 video；音频转码、语音端点检测、人声背景分离等非剪辑目标应路由到 audio。滤镜效果包括春日/晚霞/鲜亮/白皙/食物等，具体参数以工具 reference 为准。

## 工具列表

| 工具 | 说明 | 支持模式 | 命令 | 参考 |
| --- | --- | --- | --- | --- |
| add-image-to-video | 支持将指定图片（如 Logo、水印等）叠加到视频画面上。 | Cloud / Local | `mediakit-cli --cloud editing add-image-to-video` / `mediakit-cli --local editing add-image-to-video` | [reference/add-image-to-video.md](reference/add-image-to-video.md) |
| add-subtitle-to-video | 将字幕文件或文本内容按自定义样式压制到视频画面中，生成带内嵌字幕的新视频。 | Cloud / Local | `mediakit-cli --cloud editing add-subtitle-to-video` / `mediakit-cli --local editing add-subtitle-to-video` | [reference/add-subtitle-to-video.md](reference/add-subtitle-to-video.md) |
| adjust-audio-speed | 用于音频调速，可调整音频播放倍速，实现快放或慢放效果。 | Cloud / Local | `mediakit-cli --cloud editing adjust-audio-speed` / `mediakit-cli --local editing adjust-audio-speed` | [reference/adjust-audio-speed.md](reference/adjust-audio-speed.md) |
| adjust-video-speed | 用于视频调速，通过调整播放倍速产生快放或慢放效果。 | Cloud / Local | `mediakit-cli --cloud editing adjust-video-speed` / `mediakit-cli --local editing adjust-video-speed` | [reference/adjust-video-speed.md](reference/adjust-video-speed.md) |
| adjust-video-volume | 用于调整输入视频的音量大小，也可实现静音。 | Cloud / Local | `mediakit-cli --cloud editing adjust-video-volume` / `mediakit-cli --local editing adjust-video-volume` | [reference/adjust-video-volume.md](reference/adjust-video-volume.md) |
| apply-camera-motion | 为静态或平稳拍摄的视频素材在指定时间段内添加缩放、平移缩放、360 度旋转或弹跳等预设运镜效果，适用于素材二次创作、营销片头和短剧动效。 | Cloud | `mediakit-cli editing apply-camera-motion` | [reference/apply-camera-motion.md](reference/apply-camera-motion.md) |
| apply-video-filter | 为视频应用预设滤镜，调整画面的色调、饱和度、对比度或人像肤色，形成清新、暖调、鲜亮等视觉氛围。 | Cloud | `mediakit-cli editing apply-video-filter` | [reference/apply-video-filter.md](reference/apply-video-filter.md) |
| concat-audio | 将多个音频片段按列表顺序首尾连接，生成一个连续的音频文件；它用于时间轴拼接，不用于让多路音频同时播放。 | Cloud / Local | `mediakit-cli --cloud editing concat-audio` / `mediakit-cli --local editing concat-audio` | [reference/concat-audio.md](reference/concat-audio.md) |
| concat-video | 将多个视频按顺序拼接成一个完整的视频文件，并支持在拼接处添加转场效果。 | Cloud / Local | `mediakit-cli --cloud editing concat-video` / `mediakit-cli --local editing concat-video` | [reference/concat-video.md](reference/concat-video.md) |
| crop-video | 按指定的矩形区域裁剪视频画面，裁剪结果仅保留指定的需要区域。 | Cloud | `mediakit-cli editing crop-video` | [reference/crop-video.md](reference/crop-video.md) |
| extract-animated-image | 从视频中按指定开始时间和结束时间截取一段画面，生成 GIF 或 WebP 动图，常用于制作封面动图、营销素材和短预览。 | Cloud | `mediakit-cli editing extract-animated-image` | [reference/extract-animated-image.md](reference/extract-animated-image.md) |
| extract-audio | 从输入视频文件中分离音轨，生成独立的音频文件。 | Cloud / Local | `mediakit-cli --cloud editing extract-audio` / `mediakit-cli --local editing extract-audio` | [reference/extract-audio.md](reference/extract-audio.md) |
| fade-audio | 对输入音频的起止位置实现淡入或淡出效果，输出处理后的音频文件。 | Cloud / Local | `mediakit-cli --cloud editing fade-audio` / `mediakit-cli --local editing fade-audio` | [reference/fade-audio.md](reference/fade-audio.md) |
| fade-video-audio | 在片头或片尾对输入视频音轨执行淡入或淡出处理，用于弱化音轨突兀的起止，提升成片听感。输出处理后的视频文件。 | Cloud / Local | `mediakit-cli --cloud editing fade-video-audio` / `mediakit-cli --local editing fade-video-audio` | [reference/fade-video-audio.md](reference/fade-video-audio.md) |
| flip-video | 用于视频画面翻转，对指定视频进行上下或左右镜像翻转。 | Cloud / Local | `mediakit-cli --cloud editing flip-video` / `mediakit-cli --local editing flip-video` | [reference/flip-video.md](reference/flip-video.md) |
| image-to-video | 将多张图片按顺序组合成动态视频，可配置转场动画和镜头内动画；仅把现有图片做成带动效的视频，不支持根据参考图生成新的画面内容。 | Cloud | `mediakit-cli editing image-to-video` | [reference/image-to-video.md](reference/image-to-video.md) |
| mix-audio | 将多个音频文件同时混合为一个新的音频文件，适合把人声、背景音乐和音效叠加播放；输出时长以最长的输入音频为准。 | Cloud / Local | `mediakit-cli --cloud editing mix-audio` / `mediakit-cli --local editing mix-audio` | [reference/mix-audio.md](reference/mix-audio.md) |
| mux-audio-video | 可将输入的音频流与视频流合并成一个新的视频文件，并可选择保留或替换视频的原有音轨；当音视频时长不一致时，可进行对齐处理。 | Cloud / Local | `mediakit-cli --cloud editing mux-audio-video` / `mediakit-cli --local editing mux-audio-video` | [reference/mux-audio-video.md](reference/mux-audio-video.md) |
| rotate-video | 用于视频画面旋转，对指定视频进行整体旋转。 | Cloud | `mediakit-cli editing rotate-video` | [reference/rotate-video.md](reference/rotate-video.md) |
| stitch-video | 将 2 至 3 个视频在同一画布中按左右或上下方向进行空间拼接，适用于多视角对比和画面组合；输出时长以最长输入为准，较短视频结束后以黑屏补齐，并混合所有选择保留的音轨。 | Cloud | `mediakit-cli editing stitch-video` | [reference/stitch-video.md](reference/stitch-video.md) |
| text-to-scrolling-video | 将指定文本内容转换为文字滚屏视频，输出视频为固定 9:16 竖版，常用于小说推文、内容讲解和歌词视频等场景。 | Cloud | `mediakit-cli editing text-to-scrolling-video` | [reference/text-to-scrolling-video.md](reference/text-to-scrolling-video.md) |
| trim-audio | 用于音频裁剪，按指定的开始时间和结束时间从输入音频中截取片段。 | Cloud / Local | `mediakit-cli --cloud editing trim-audio` / `mediakit-cli --local editing trim-audio` | [reference/trim-audio.md](reference/trim-audio.md) |
| trim-video | 用于视频裁剪，可按指定的开始和结束时间从输入视频截取片段。 | Cloud / Local | `mediakit-cli --cloud editing trim-video` / `mediakit-cli --local editing trim-video` | [reference/trim-video.md](reference/trim-video.md) |
