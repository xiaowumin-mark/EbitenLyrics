# 计划六 A：视觉回归清单与基线样例

目标：对照 `ref/applemusic-like-lyrics/packages/core` 建立可重复的视觉回归标准。计划六后续 B～F 的调参都必须先回到本清单，避免只凭单次观感随意改参数。

## 参考依据

### 时间线与滚动

参考文件：

- `ref/applemusic-like-lyrics/packages/core/src/lyric-player/base/timeline.ts`
- `ref/applemusic-like-lyrics/packages/core/src/lyric-player/base/index.ts`
- `ref/applemusic-like-lyrics/packages/core/src/lyric-player/base/scroll.ts`

基线行为：

- `hotLines` 表示当前时间命中的行；`bufferedLines` 表示 UI 上仍保持激活或过渡显示的行。
- 非 seek 状态下，新增 hot line 会加入 buffered；只有 buffered 全部移除或加入新行时才改变布局。
- seek 状态会直接重建 buffered，并用 `pickScrollToIndexForSeek` 选择对齐目标。
- 播放到末尾且 bottom line 有内容时，`scrollToIndex == len(lines)`，bottom line 成为对齐目标。
- 布局入口是 `calcLayout`，每次只更新目标位置；实际运动由每帧 `update` 推进 spring。
- 用户滚动只修改 `scrollState.scrollOffset`，并通过边界 clamp，不应整体平移歌词画布。
- 用户滚动后 `isScrolled` 保持约 5 秒，然后重置 `scrollOffset = 0`。

### 布局与 spring

参考文件：

- `ref/applemusic-like-lyrics/packages/core/src/lyric-player/base/layout.ts`
- `ref/applemusic-like-lyrics/packages/core/src/lyric-player/base/index.ts`

基线行为：

- 默认 `alignAnchor = Center`，`alignPosition = 0.35`，目标行中心落在视口高度约 35% 的位置。
- 默认 `overscanPx = 300`，可视区外的行仍保留一定预渲染距离。
- 初始 Y spring 参数是 `mass: 0.9, damping: 15, stiffness: 90`。
- seeking 或 interlude active 时，Y spring 使用稳定参数 `stiffness: 90, damping: 15`。
- 普通播放时，根据当前行与上一行时间间隔计算 `stiffness: 170..220`，`damping = sqrt(stiffness) * 2.2`。
- 每行 transform 带递增 delay：基础约 `0.05s`，越靠后 delay 衰减，形成连续跟随感。
- bottom line 的 Y spring 参数和普通行同步。

### presentation 与 blur

参考文件：

- `ref/applemusic-like-lyrics/packages/core/src/lyric-player/base/layout.ts`
- `ref/applemusic-like-lyrics/packages/core/src/lyric-player/dom/lyric-line.ts`
- `ref/applemusic-like-lyrics/packages/core/src/styles/lyric-player.module.css`

基线行为：

- active 判定：`hasBuffered || (lineIndex >= scrollToIndex && lineIndex < latestIndex)`。
- buffered 行 opacity 为 `0.85`。
- 非逐词歌词的 inactive opacity 为 `0.2`；逐词歌词 inactive opacity 为 `1`。
- 启用 `hidePassedLines` 且播放中时，已播放行 opacity 为 `1e-4`；interlude 中隐藏边界为 `anchorLineIndex + 1`。
- 播放中 inactive 主行 scale 为 `0.97`；inactive 背景行 scale 为 `0.75`。
- active 行使用 gradient render mode；inactive 行使用 solid render mode。
- blur 关闭、用户滚动中、active 行时 blur 为 `0`。
- 目标行之前的行 blur 随距离更强；目标行之后的行按 `max(scrollToIndex, latestIndex)` 计算距离。
- 紧凑布局 blur 乘以 `0.8`；DOM 实际 filter clamp 到约 `5px`。

### interlude dots

参考文件：

- `ref/applemusic-like-lyrics/packages/core/src/lyric-player/base/layout.ts`
- `ref/applemusic-like-lyrics/packages/core/src/lyric-player/dom/interlude-dots.ts`
- `ref/applemusic-like-lyrics/packages/core/src/styles/lyric-player.module.css`

基线行为：

- 当前时间先加 `20ms` 再判断 interlude。
- gap 起点是上一行 `endTime`，终点是下一行 `startTime - 250ms`。
- 只有 gap 时长至少 `4000ms` 时才显示 dots。
- 检查范围是 `scrollToIndex - 1`、`scrollToIndex`、`scrollToIndex + 1`。
- dots 插入在 `anchorLineIndex + 1` 之前；如果 `anchorLineIndex != -1`，布局起始位置先扣除 dots 总高度。
- dots margin 是 `fontSize * 0.4`。
- dots 前 500ms 透明，500～1000ms fade in。
- dots 前 2000ms 使用 `easeOutExpo` 进入 scale。
- 最后 750ms 开始 exit scale；最后 375ms fade out。
- 三个 dots 亮度在 `interludeDuration - 750ms` 内依次从 `0.25` 到 `1`。
- 下一行为 duet 时，dots 靠右对齐。

### bottom line

参考文件：

- `ref/applemusic-like-lyrics/packages/core/src/lyric-player/base/bottom-line.ts`
- `ref/applemusic-like-lyrics/packages/core/src/lyric-player/base/index.ts`
- `ref/applemusic-like-lyrics/packages/core/src/styles/lyric-player.module.css`

基线行为：

- bottom line 是歌词列表末尾的扩展槽，不是普通歌词行。
- bottom line 空时不占视觉内容；有内容时参与末尾布局和滚动边界。
- `scrollToIndex == len(lines)` 时 bottom line focused，blur 为 `0`。
- bottom line transform 与 blur 在 `calcLayout` 末尾设置，Y spring 与普通歌词同步。
- ref 推荐该扩展槽用于显示歌曲创作者等信息，但 core 不强制具体 UI。

## 回归样例

每个样例都要在以下基础配置下观察一次：

- 窗口尺寸：桌面宽屏，至少 `1280x720`。
- 播放状态：正常播放、暂停、seek 后恢复。
- blur：默认开启，并确认用户滚动中 blur 关闭。
- debug panel：记录 `歌词模糊强度` 当前值。
- 如果使用 WebSocket，优先按实际顺序 `setMusic -> setLyric -> progress` 回放。

### 样例 A：普通逐词歌词

用途：确认主路径是否稳定。

输入特征：

- 每行有多个 words，时间间隔中等。
- 不含背景行、不含 duet、不含长间奏。

观察点：

- 当前行中心应落在视口高度约 35% 处。
- 新行进入时，旧 buffered 行不会立刻消失，滚动锚点不应抖动。
- active 行 scale 接近 `1`，inactive 主行 scale 接近 `0.97`。
- active 行为 gradient，高亮逐词推进；inactive 行为 solid。
- blur 随距离增加，但 active / buffered 区域清晰。

通过标准：

- 连续 10 行播放中没有明显跳帧式滚动、行重叠或突然消失。
- seek 到任意行后，布局能直接落到合理位置，再继续自然滚动。

### 样例 B：快速连唱 / 短间隔歌词

用途：验证动态 spring 参数与 per-line delay。

输入特征：

- 相邻主行 start 间隔约 `100ms～450ms`。
- 至少连续 8 行快速切换。

观察点：

- Y spring 应更紧，滚动跟随更快，不应拖到下一句已唱完才到位。
- per-line delay 仍应有波浪跟随感，但不能造成行列整体“橡皮筋过长”。
- bufferedLines 切换时不应导致 active 行 alpha 闪烁。

通过标准：

- 快速段落中当前唱到的行始终保持在视觉焦点附近。
- 快速切换后回到慢速歌词时，spring 观感能恢复稳定，不残留过强速度。

### 样例 C：长间奏 dots

用途：验证 interlude 计算、dots 动画和布局占位。

输入特征：

- 两句主行之间 gap 至少 `4000ms`。
- 下一行分别测试普通行和 duet 行。

观察点：

- dots 只在 gap 中出现，且结束于下一行开始前约 `250ms`。
- dots 进入：前 500ms 不可见，随后 500ms fade in，前 2000ms scale 进入。
- dots 退出：最后 750ms scale 收缩，最后 375ms fade out。
- dots 与歌词之间的上下 margin 接近 `fontSize * 0.4`。
- 下一行为 duet 时 dots 靠右，不是固定在普通行左边。

通过标准：

- dots 不跳到左上角、不残留到下一句 active 后、不在短 gap 中误出现。
- 用户滚动或 seek 后，dots 的当前时间不漂移。

### 样例 D：背景行与 pause/resume

用途：验证 background line 的显示、占位和播放状态联动。

输入特征：

- 主行带背景行，背景行可能和主行时间重叠。
- 包含暂停和恢复操作。

观察点：

- 播放中 inactive 背景行 scale 接近 `0.75`，通常不抢主行焦点。
- 背景行 active 或暂停时应可见，opacity 约 `0.4` 的观感。
- 背景行是否 reserve space 要和 presentation 结果一致，不应突然压缩主行。
- pause 后背景行显示稳定；resume 后回到播放规则。

通过标准：

- 暂停/恢复不会让背景行突然跳位置、消失或永久占位错误。
- 主行与背景行在快速切换时不互相覆盖。

### 样例 E：duet / 右侧歌词

用途：验证 X 轴布局、duet avoidance 和 dots 右对齐。

输入特征：

- 歌曲中至少有一行 duet。
- 同时包含普通行和 duet 行。
- 可选：duet 行前有长间奏。

观察点：

- 普通行和 duet 行都使用全宽容器，但 padding 不同。
- duet 行文本应明显靠右，普通行应避免和 duet 区域冲突。
- interlude dots 若指向 duet 下一行，应靠右。

通过标准：

- 普通行不被错误推到右侧；duet 行不挤压到普通行左侧。
- resize 后左右 padding 和行宽能重新计算。

### 样例 F：歌曲末尾 BottomLine

用途：验证 bottom line 扩展槽、末尾滚动边界和 `setMusic` 状态。

输入特征：

- WebSocket 先发 `setMusic`，再发 `setLyric`。
- `setMusic.artists` 至少包含一个 `name`。
- 歌词播放到最后一行之后。

观察点：

- BottomLine 显示 `创作者：...`，左边与普通歌词文本对齐。
- `setMusic -> setLyric` 后 bottom line 不丢失。
- 歌曲结束后，如果 bottom line 有内容，滚动目标应为 `len(lines)`。
- bottom focused 时 blur 为 `0`，focus alpha 更高。
- 无 artist 的新 `setMusic` 应清空 bottom line。

通过标准：

- 播放到末尾时 bottom line 能自然进入焦点，不突然截断滚动边界。
- 字体切换、窗口 resize 后文本仍清晰且位置正确。

### 样例 G：用户滚动与自动回归

用途：验证 scrollState 与 presentation 的联动。

输入特征：

- 任意中长歌词，至少 20 行。
- 播放中执行滚轮滚动。

观察点：

- 滚轮只改变 `scrollOffset`，不应造成整层画面和时间线脱节。
- 滚动中 blur 应关闭，行可读性提升。
- 滚动范围受 `ScrollMinOffset / ScrollMaxOffset` 限制，不越界。
- 约 5 秒后自动 reset 到当前播放目标。

通过标准：

- 手动滚动中歌词不越界，不因为自动 progress 更新而抢回位置。
- reset 后继续播放时，滚动与 active 行同步。

## 手动回归流程

1. 记录当前分支、提交、窗口尺寸、字体、blur strength、是否开启 hidePassedLines。
2. 启动应用并确认 `go test ./...` 在调参前通过。
3. 按样例 A～G 逐项播放或 seek。
4. 每个样例记录：通过 / 有轻微差异 / 阻塞问题。
5. 若要调参，只允许一次修改一个参数组，例如只调 spring 或只调 dots margin。
6. 调参后重复受影响样例，不要只看当前失败场景。
7. 调参完成后运行 `go test ./lyrics`、`go test ./pages`、`go test ./...`。
8. 把最终参数和仍可接受差异回写到 `docs/lyrics-component-parity-plan.md`。

## 可接受差异

- Ebiten 文本抗锯齿、混合模式和 DOM/CSS `filter` 不可能完全一致，只要求高层状态和运动节奏接近。
- BottomLine 当前 C1 是通用单行文本槽，不要求和 DOM 插槽中的任意 HTML 内容能力一致。
- 当前不要求触摸惯性完全复刻浏览器实现；桌面滚轮、边界 clamp、5 秒 reset 必须稳定。
- 当前不做像素级截图测试；计划六测试优先覆盖状态、边界、缓存和参数函数。

## 6A 完成标准

- [x] 明确 ref 依据与行为基线。
- [x] 覆盖至少 5 类回归样例；当前覆盖 A～G 共 7 类。
- [x] 每类样例包含输入特征、观察点和通过标准。
- [x] 明确手动回归流程和可接受差异。
- [ ] 后续 B～F 调参结果回写到本文档或 parity plan。
