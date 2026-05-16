# 歌词组件显示效果对齐计划

目标：在不重写 Ebiten 底层渲染方式的前提下，让当前项目的歌词组件在显示层面尽量接近 `ref/applemusic-like-lyrics/packages/core` 的 Apple Music 风格效果。

## 总体方向

- 保留现有 Ebiten 绘制、字体、SyllableImage、渐变高亮、文本阴影等底层能力。
- 新增播放器状态、布局状态、presentation 状态，让动画由状态驱动。
- 逐步替换 `nowLyrics + ScrollLyrics + LineAnimate + NormalizeLine` 这种分散逻辑。
- 优先复刻 ref 的视觉行为：缓冲行、动态弹簧滚动、非激活行缩放/透明度/模糊、背景歌词占位策略、歌词预处理。

## 当前主要差异

### 1. 时间线状态

- ref 使用 `hotLines` 表示当前时间命中的行。
- ref 使用 `bufferedLines` 表示 UI 上仍保持激活或过渡显示的行。
- ref 使用 `scrollToIndex` 单独决定滚动目标。
- 当前项目主要使用 `nowLyrics` 表示当前命中行，离开后直接删除，再通过 `NormalizeLine` 做退出动画。
- 影响：当前实现更容易出现行切换突兀、退场太快、滚动锚点不稳定等问题。

### 2. 滚动机制

- ref 的 `calcLayout` 只计算每行目标位置，实际位置由每行 `Spring` 持续逼近。
- 当前项目每次 `ScrollLyrics` 给每行创建 `ScrollAnimate` tween。
- 影响：ref 的滚动更连续、更有惯性，重定向更自然；当前项目快速歌词、seek、背景行变化时更容易机械或断点明显。

### 3. Presentation 计算

- ref 用 `computeLinePresentation` 统一计算每行是否 active、目标 opacity、scale、blur、renderMode。
- 当前项目把 scale、alpha、active、background 显示逻辑分散在 `LineAnimate`、`NormalizeLine`、`ScrollLyrics`、`backgroundLineReservesSpace`。
- 影响：视觉策略难统一，背景行退出、主行高亮衰减、静态层缓存状态更容易出现边缘问题。

### 4. 弹簧参数

- ref 会根据行间隔动态调整纵向滚动弹簧参数。
- 快速连唱时弹簧更紧，长间隔时更稳。
- 当前项目使用 normal / fast 两套 tween duration 和 easing。
- 影响：当前项目能做到快慢不同，但缺少 Apple Music 那种连续物理感。

### 5. 歌词预处理

- ref 加载歌词时会做 `optimizeLyricLines`：空格规范化、行时间戳修正、主/背景行同步、清理非刻意重叠、尝试提前行开始时间。
- 当前项目只有部分 TTML 处理和 `maxLineEndWithBackground`。
- 影响：行进入时机、滚动提前感、背景行同步会有差距。

### 6. 行间奏

- ref 会在长空白区显示 interlude dots，并在布局中给 dots 占位。
- 当前项目没有等价的间奏显示和布局状态。
- 影响：长间隔处缺少 Apple Music 的呼吸感和等待反馈。

## 推荐实施顺序

### 阶段 1：新增状态层

新增类似 ref 的状态结构，建议放在 `lyrics/type.go` 或新文件。

建议新增：

- `TimelineState`
- `LayoutState`
- `ScrollState`
- `LinePresentation`

`TimelineState` 建议包含：

- `CurrentTime time.Duration`
- `LastCurrentTime time.Duration`
- `HotLines map[int]struct{}`
- `BufferedLines map[int]struct{}`
- `ScrollToIndex int`
- `IsSeeking bool`
- `IsPlaying bool`
- `InitialLayoutFinished bool`

`LayoutState` 建议包含：

- `TargetAlignIndex int`
- `AlignPosition float64`，默认可用 `0.35`
- `AlignAnchor`，第一版可只支持 center
- `Overscan float64`
- `LastInterludeActive bool`

第一阶段可以保留 `nowLyrics` 作为兼容字段，后续再逐步替换。

### 阶段 2：移植时间线算法

参考 `ref/applemusic-like-lyrics/packages/core/src/lyric-player/base/timeline.ts`，在 Go 中实现纯函数：

- `computePlayerTimeState`
- `commitPlayerTimeState`
- seek 场景下的 scroll target 计算

当前 `UpdateLyrics` 应逐步改为：

- 根据当前时间计算 next hot lines。
- 得到 `linesToEnable` 和 `linesToDisable`。
- enable 行只触发行内动画开始。
- disable 行只触发行内动画退出。
- 如果需要布局，调用新的 `CalcLayout`。

seek 时应走特殊路径：

- 直接重算 hot/buffered。
- 直接更新 `scrollToIndex`。
- 允许 force layout，避免拖动进度时慢慢滚过去。

### 阶段 3：新增 Presentation 层

参考 `ref/applemusic-like-lyrics/packages/core/src/lyric-player/base/layout.ts` 的 `computeLinePresentation`。

Go 中建议新增：

- `ComputeLinePresentation(input LinePresentationInput) LinePresentation`

每行统一计算：

- `IsActive`
- `TargetAlpha`
- `TargetScale`
- `BlurLevel`
- `RenderMode`

第一版策略建议贴近 ref：

- buffered 行：`alpha = 0.85`
- 非动态歌词未激活：`alpha = 0.2`
- 普通未激活主行：`scale = 0.97`
- 背景未激活行：`scale = 0.75`
- active 行：`scale = 1`
- active 行使用 gradient / 高亮模式
- 非 active 行使用 solid / 低亮模式

### 阶段 4：重写布局驱动

把 `scrollLyricsTo` 改造成 ref 风格的 `CalcLayout`：

- 以 `scrollToIndex` 为锚点。
- 计算每行目标 Y。
- 计算 render set / overscan。
- 计算 presentation。
- 不在布局函数里直接决定 active enter/exit。

`ScrollLyrics` 可以先保留为兼容包装。

### 阶段 5：引入 Spring 动画

在 `anim` 中新增 `Spring` 类型，参考 ref 的 `utils/spring.ts`。

建议能力：

- `SetPosition(value)`：立即设置当前位置和目标位置。
- `SetTargetPosition(value)`：保留当前速度，重定向到新目标。
- `UpdateParams(params)`：更新 mass、damping、stiffness。
- `Update(dt)`：每帧推进。
- `CurrentPosition()`：读取当前位置。

每个 `Line` 可新增：

- `PosYSpring`
- `ScaleSpring`
- 可选 `AlphaSpring`
- 可选 `BlurSpring`

第一版建议只替换 Y 和 scale：

- `CalcLayout` 只设置目标 Y 和目标 scale。
- 每帧更新 spring，并写回 `Position.Y`、`ScaleX`、`ScaleY`。
- 现有 `ScrollAnimate` 暂时保留作为 fallback。

### 阶段 6：动态弹簧参数

参考 ref 的 `computeLinePosYSpringParams`：

- seeking 或 interlude 时：`stiffness = 90`，`damping = 15`
- 普通播放时根据当前行和上一行时间间隔动态计算。

建议映射：

- interval clamp 到 `100ms ~ 800ms`
- stiffness 映射到 `170 ~ 220`
- damping 使用 `sqrt(stiffness) * 2.2`

每次 `scrollToIndex` 变化时更新所有行的 Y Spring 参数。

### 阶段 7：整理 enter / exit 职责

`EnableLine` 只负责：

- 设置 active 状态。
- 启动逐字 mask / word float / emphasize。

`DisableLine` 只负责：

- 停止或反向处理逐字动画。
- 让 highlight 退场。

行整体的 scale、alpha、y 不再由 enter/exit tween 管，而由 presentation + spring 管。

### 阶段 8：补歌词 optimize

参考 `ref/applemusic-like-lyrics/packages/core/src/utils/optimize-lyric.ts`。

建议新增 `lyrics/optimize.go` 或 `ttml/optimize.go`。

优先实现：

- `normalizeSpaces`
- `resetLineTimestamps`
- `syncMainAndBackgroundLines`
- `cleanUnintentionalOverlaps`
- `tryAdvanceStartTime`

其中 `tryAdvanceStartTime` 很关键：

- 默认尝试提前 `600ms`。
- 如果和上一行重叠，fallback 提前 `400ms` 或上一行时长的 `30%`。

### 阶段 9：背景行 / 对唱行策略对齐

背景行改造方向：

- 背景行也纳入 presentation 计算。
- `backgroundLineReservesSpace` 不再只看 `Status`，而看 presentation 的 `IsActive` 或 `IsPlaying == false`。
- 背景行 alpha / scale 目标由 presentation 给出。

对唱行：

- 保留当前 `IsDuet` 和右对齐逻辑。
- 后续再补 ref 的 dots/right alignment 行为。

### 阶段 10：补 blur 和 interlude dots

blur 建议后做，因为有性能风险。

blur 路线：

- 给 `Line` 加 `BlurLevel`。
- `computeLineBlur` 计算目标值。
- 第一版可用 alpha 模拟距离层次。
- 性能允许后再接 `filters.BlurImageShader`。

interlude dots 路线：

- 新增 `Interlude` 结构：`StartTime`、`EndTime`、`AnchorLineIndex`、`IsNextDuet`。
- 实现 `computeCurrentInterlude`。
- gap 大于 `4000ms` 时显示 dots。
- 下一行开始前预留 `250ms`。
- 布局中为 dots 插入额外高度。

## 第一版建议目标

第一版不要一次性追求 100%。建议先做到：

- hot/buffered/scrollToIndex 状态。
- optimize 行时间。
- presentation 统一 scale / alpha。
- Y 轴 spring 滚动。
- scale spring。
- 暂不做 blur。
- 暂不做 interlude dots。

这一版应优先改善：

- 当前行切换突兀。
- 滚动机械感。
- 快歌滚动不自然。
- 背景歌词顶布局。
- 行进入偏晚或偏早。
- 非当前行层次感不足。

## 推荐路线

推荐采用稳妥路线：

1. 先移植 timeline / optimize / presentation。
2. 再做 Spring 滚动。
3. 最后补 blur 和 interlude dots。

原因：先建立清晰状态和目标值，后续 Spring、blur、dots 都能挂在统一状态上，避免继续让动画逻辑分散。
