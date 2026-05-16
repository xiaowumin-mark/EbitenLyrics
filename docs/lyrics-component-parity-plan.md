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

## 早期总体路线

本节是最初拆分出的总体技术路线，阶段编号保留用于追溯设计依据。当前实际开发已按后续“计划一～计划四”推进；其中 Spring 动画相关内容已经归入并完成于“计划二”，不要再把本节的“阶段 5”理解为下一步任务。

当前实际计划进度见文末“当前计划进度”。

## 推荐实施顺序

### 阶段 1：新增状态层

状态：已完成，归入计划一。

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

状态：已完成，归入计划一。

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

状态：已完成，归入计划一与计划三。

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

状态：已完成，归入计划二。

把 `scrollLyricsTo` 改造成 ref 风格的 `CalcLayout`：

- 以 `scrollToIndex` 为锚点。
- 计算每行目标 Y。
- 计算 render set / overscan。
- 计算 presentation。
- 不在布局函数里直接决定 active enter/exit。

`ScrollLyrics` 可以先保留为兼容包装。

### 阶段 5：引入 Spring 动画

状态：已完成，归入计划二。当前不要把该阶段视为下一步计划。

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

状态：已完成，归入计划二与计划四 BottomLine spring 收口。

参考 ref 的 `computeLinePosYSpringParams`：

- seeking 或 interlude 时：`stiffness = 90`，`damping = 15`
- 普通播放时根据当前行和上一行时间间隔动态计算。

建议映射：

- interval clamp 到 `100ms ~ 800ms`
- stiffness 映射到 `170 ~ 220`
- damping 使用 `sqrt(stiffness) * 2.2`

每次 `scrollToIndex` 变化时更新所有行的 Y Spring 参数。

### 阶段 7：整理 enter / exit 职责

状态：已完成，归入计划二。

`EnableLine` 只负责：

- 设置 active 状态。
- 启动逐字 mask / word float / emphasize。

`DisableLine` 只负责：

- 停止或反向处理逐字动画。
- 让 highlight 退场。

行整体的 scale、alpha、y 不再由 enter/exit tween 管，而由 presentation + spring 管。

### 阶段 8：补歌词 optimize

状态：已完成，归入计划一。

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

状态：已完成，归入计划三的 presentation、X 轴布局和 pause/resume 联动。

背景行改造方向：

- 背景行也纳入 presentation 计算。
- `backgroundLineReservesSpace` 不再只看 `Status`，而看 presentation 的 `IsActive` 或 `IsPlaying == false`。
- 背景行 alpha / scale 目标由 presentation 给出。

对唱行：

- 保留当前 `IsDuet` 和右对齐逻辑。
- 后续再补 ref 的 dots/right alignment 行为。

### 阶段 10：补 blur 和 interlude dots

状态：interlude dots 已完成，归入计划三；真实 blur 仍暂缓，候选为后续计划五。

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
3. 最后补完整 presentation、interlude dots 和用户滚动状态；blur 暂时只保留接口，不接真实模糊。

原因：先建立清晰状态和目标值，后续 Spring、dots、blur 都能挂在统一状态上，避免继续让动画逻辑分散。

## 计划三：显示层次与间奏呼吸感

计划三目标：在不接入真实 blur shader 的前提下，继续对齐 `ref/applemusic-like-lyrics/packages/core` 的上层显示行为，让歌词具备更接近 Apple Music 的空间层次、间奏呼吸感和用户滚动状态联动。

本阶段明确暂不做真实 blur 渲染：`BlurLevel` 可以先作为 presentation 结果保留，但渲染层不调用 `filters.BlurImageShader`，避免性能和布局问题混在一起。

### 阶段 3.1：补全 Presentation 输入与规则

参考 `ref/applemusic-like-lyrics/packages/core/src/lyric-player/base/layout.ts` 的 `computeLinePresentation` 和 `computeLineBlur`。

扩展当前 `linePresentationInput`：

- `HidePassedLines bool`
- `EnableBlur bool`
- `IsUserScrolling bool`
- `IsCompact bool`
- `Interlude *Interlude`

扩展后的 presentation 行为：

- active 行：`scale = 1`，`blur = 0`。
- buffered 行：`alpha = 0.85`，`scale = 1`。
- inactive 主行：`alpha = 1`，`scale = 0.97`。
- inactive 背景行：`scale = 0.75`。
- 非动态歌词 inactive：`alpha = 0.2`。
- `HidePassedLines` 开启时，已播放行接近不可见：`alpha = 1e-4`。
- 用户滚动中，`BlurLevel = 0`。
- blur 暂时只计算数值，不接真实绘制。

第一步只接 alpha / scale / reserve space，`BlurLevel` 作为后续保留字段。

### 阶段 3.2：实现 Interlude 数据层

新增结构：

```go
type Interlude struct {
    StartTime       time.Duration
    EndTime         time.Duration
    AnchorLineIndex int
    IsNextDuet      bool
}
```

新增函数：

```go
func computeCurrentInterlude(l *Lyrics) *Interlude
```

对齐 ref 的判断规则：

- 使用 `currentTime + 20ms` 判断当前时间。
- 检查 `scrollToIndex - 1`、`scrollToIndex`、`scrollToIndex + 1` 附近的 gap。
- gap 起点为上一行结束时间；第一行前则为 `0`。
- gap 终点为下一行开始时间前 `250ms`。
- gap 小于 `4000ms` 不显示。
- 当前时间处于 gap 内才返回 interlude。
- `AnchorLineIndex = -1` 表示 dots 位于第一行之前。
- `IsNextDuet` 用下一行的 `IsDuet` 决定 dots 靠左还是靠右。

接入点：

- 在 `scrollLyricsTo` / `calcLayout` 开头计算 interlude。
- 如果 interlude active，`computeLinePosYSpringParams` 使用稳定参数。
- `LayoutState.LastInterludeState` 用于判断 interlude 状态变化时是否需要更新 spring 参数。

### 阶段 3.3：布局中插入 Interlude Dots 占位

先让 dots 参与布局，再实现绘制。

对齐 ref 的布局语义：

- dots 有自己的高度和上下 margin。
- 如果 interlude 存在且 `AnchorLineIndex != -1`，初始 `curPos` 需要提前减去 dots 总高度。
- 主循环中，当 `i == AnchorLineIndex + 1` 时插入 dots：
  - `curPos += dotMargin`
  - 设置 dots 目标位置
  - `curPos += dotsHeight`
  - `curPos += dotMargin`
- 如果 `AnchorLineIndex == -1`，dots 插在第一行之前。
- 对唱下一行时，dots 靠右；否则靠左。

建议尺寸：

- `dotSize = fontSize * 0.14`
- `dotGap = fontSize * 0.18`
- `dotMargin = fontSize * 0.4`
- `dotsHeight = dotSize * 1.4`

### 阶段 3.4：实现 InterludeDots 绘制组件

新增轻量组件：

```go
type InterludeDots struct {
    Position Position
    Active bool
    Paused bool
    Phase float64
    StartTime time.Duration
    EndTime time.Duration
}
```

行为：

- 3 个圆点横向排列。
- 每个点使用错相位呼吸动画。
- alpha 在 `0.25 ~ 1` 之间变化。
- scale 在 `0.85 ~ 1.12` 之间变化。
- pause 时 phase 停止。
- resume 后继续。

接入点：

- `Lyrics` 持有 `InterludeDots`。
- `Lyrics.Tick(dt)` 推进 dots phase。
- `DrawDynamic` 或统一 draw 流程中绘制 dots。
- `scrollLyricsTo` 设置 dots 的位置和 active 时间范围。

### 阶段 3.5：接入用户滚动状态

当前页面已有用户滚动状态：`Home.isUserScrolling`、`manualScrollOffset`、`manualScrollTarget`。

计划新增接口：

```go
func (l *Lyrics) SetUserScrolling(scrolling bool)
func (l *LyricsComponent) SetUserScrolling(scrolling bool)
```

状态建议放入 `LayoutState`：

- `IsUserScrolling bool`
- 可选 `HidePassedLines bool`
- 可选 `EnableBlur bool`

接入行为：

- `Home.beginUserScroll()` 调用 `LyricsControl.SetUserScrolling(true)`。
- 手动滚动结束后调用 `SetUserScrolling(false)` 并触发布局。
- 用户滚动中 presentation 强制 `BlurLevel = 0`。
- 用户滚动中暂不隐藏 passed lines，避免滚动查看历史歌词时上方内容消失。

### 阶段 3.6：暂停/播放状态联动

对齐 ref 的 pause/resume 行为：

- `Pause()` 时 `Timeline.IsPlaying = false`，重新布局。
- 暂停时背景歌词可以显示并占位。
- `Resume()` 时 `Timeline.IsPlaying = true`，重新布局。
- dots pause 时停止呼吸动画，resume 后继续。

当前已存在 `Lyrics.Pause()` / `Lyrics.Resume()`，计划三需要确保 presentation 和 interlude dots 完整读取该状态。

### 阶段 3.7：测试计划

新增测试建议：

- `lyrics/interlude_test.go`
  - gap 小于 `4000ms` 不显示。
  - gap 大于等于 `4000ms` 显示。
  - 第一行前 interlude 正确返回 `AnchorLineIndex = -1`。
  - gap 结束前 `250ms` 停止显示。
  - 下一行为 duet 时 `IsNextDuet = true`。

- `lyrics/presentation_test.go`
  - active 行 `BlurLevel = 0`。
  - 用户滚动时 `BlurLevel = 0`。
  - `HidePassedLines` 下已播放行 `alpha = 1e-4`。
  - inactive 背景行 `scale = 0.75`。
  - 非动态 inactive 行 `alpha = 0.2`。

- `lyrics/interlude_dots_test.go`
  - `Tick` 推进 dots phase。
  - pause 后 phase 不变。
  - resume 后 phase 继续。

### 计划三第一版完成标准

- [x] presentation 规则完整对齐 ref 的 alpha / scale / reserve space 语义。
- [x] `BlurLevel` 已计算但不接真实 blur 渲染。
- [x] 长间奏时 dots 出现，并且布局为 dots 留出空间。
- [x] 暂停时背景歌词显示逻辑稳定。
- [x] 用户滚动状态能传入 presentation，滚动中不启用 blur / passed-line 隐藏。
- [x] `go test ./...` 通过。

### 暂缓内容

- 真实 blur shader 接入暂缓。
- bottom line / 歌曲信息区域放入计划四。
- 更复杂的用户滚动惯性边界暂缓；当前先复用 ref 风格的 `ScrollOffset` / `ScrollBoundary` 状态。

## 计划四：BottomLine 与末尾滚动边界

计划四目标：参考 `ref/applemusic-like-lyrics/packages/core` 的 `BottomLineEl`，补齐歌词列表末尾的特殊底栏元素，使末尾滚动、底部留白和后续歌曲信息扩展更接近 ref。真实 blur 仍然暂缓，只保留 `BlurLevel` 计算入口。

### 阶段 4.1：新增 BottomLine 数据结构

参考 `ref/applemusic-like-lyrics/packages/core/src/lyric-player/base/bottom-line.ts`。

建议新增：

```go
type BottomLine struct {
    Position Position
    PosYSpring *anim.Spring
    PosXSpring *anim.Spring
    BlurLevel float64
    Focused bool
    Active bool
    Text string
    LineSize [2]float64
}
```

第一版只做空 bottom line：

- 默认无文本内容。
- 默认高度为 `0`。
- 不参与实际绘制。
- 只作为布局和滚动边界的一部分存在。

后续如果需要展示歌曲信息、作者、底部提示，再在该结构上补文本渲染。

### 阶段 4.2：Lyrics 持有 BottomLine

在 `Lyrics` 中增加：

```go
BottomLine BottomLine
```

初始化：

- `newBottomLine(fontSize, width)`。
- 默认 `Active = false`，`LineSize = [width, 0]`。

Resize：

- 更新 bottom line 宽度。
- 如果有内容，重新测量高度。
- 空 bottom line 保持高度 `0`。

Tick：

- 和普通歌词行一样更新 bottom line 的 Y spring。
- 第一版可以只更新 Y spring，不绘制。

### 阶段 4.3：支持 bottom line 作为布局锚点

ref 中 `scrollToIndex == len(lines)` 表示当前焦点是 bottom line：

```ts
const bottomIndex = this.currentLyricLineObjects.length;
const isBottomFocused = targetAlignIndex === bottomIndex;
```

当前项目需要允许：

- `Timeline.ScrollToIndex == len(Lines)`。
- `scrollLyricsTo(... anchorIndex == len(Lines))`。
- `pickScrollToIndexForSeek` 返回 `len(lines)` 时不被 clamp 到最后一行。

需要修改的逻辑：

- `anchorIndex` clamp 从 `len(lines)-1` 改为允许 `len(lines)`。
- `targetLineHeight` 在 bottom focused 时使用 `BottomLine.LineSize[1]`。
- `activeSet` 不应把 bottom index 加入真实歌词行集合。
- `renderSet` 只能包含真实歌词行 index，不能包含 bottom index。

### 阶段 4.4：BottomLine 接入 calcLayout

在 `scrollLyricsTo` 的主循环结束后，参考 ref：

```ts
const bottomIndex = this.currentLyricLineObjects.length;
const finalBottomBlur = computeLineBlur(...);
this.bottomLine.setTransform(0, curPos, finalBottomBlur, force, delay);
```

Go 侧第一版建议：

- `bottomIndex := len(l.Lines)`。
- `isBottomFocused := anchorIndex == bottomIndex`。
- `BottomLine.Focused = isBottomFocused`。
- `BottomLine.BlurLevel = computeLineBlur(...)`，但不接真实 blur。
- `BottomLine.SetTransform(0, curPos, force, delay)`。
- 空 bottom line 高度为 0，因此不会额外撑高布局，但可以参与末尾锚点和边界计算。

### 阶段 4.5：修正滚动边界

ref 在布局末尾计算：

```ts
scrollBoundary.maxOffset = curPos + scrollOffset - viewportHeight / 2;
```

计划四要确保：

- `ScrollMinOffset` 包含目标行之前所有内容高度。
- `ScrollMaxOffset` 使用 bottom line 之后的 `curPos`。
- 当 bottom line 后续有内容时，滚动边界自然包含该内容。
- 用户滚动时不会在最后一行附近突然截断。

### 阶段 4.6：BottomLine Spring

第一版只需要 Y 轴 spring：

- 新增 `ensureBottomLineSprings`。
- 新增 `setBottomLineTransform`。
- `Lyrics.Tick(dt)` 更新 bottom line spring。
- `updateLinePosYSpringParams` 同步更新 bottom line Y spring 参数。

如果 bottom line 没有内容，绘制阶段仍可跳过。

### 阶段 4.7：hidePassedLines 配置入口

目前项目已有：

- `Layout.HidePassedLines`
- `SetHidePassedLines(enabled)`
- presentation 中已播放行 `alpha = 1e-4`。
- 间奏期间的隐藏边界需要参考 ref 使用 `interlude.AnchorLineIndex + 1`，而不是总是使用 `scrollToIndex`。

计划四只补 API/文档，不急着加 UI：

- 保持 `Lyrics.SetHidePassedLines`。
- 保持 `LyricsComponent.SetHidePassedLines`。
- `SetHidePassedLines` 触发布局重算，和 ref 的 `setHidePassedLines(hide) { this.hidePassedLines = hide; this.calcLayout(); }` 对齐。
- 不新增快捷键，避免调试键过多。
- 文档注明该能力已有 API，后续可接入配置文件或 debug panel。

### 阶段 4.8：测试计划

测试覆盖状态：

- `lyrics/bottom_line_test.go`
  - [x] 默认 bottom line 高度为 `0`。
  - [x] `scrollLyricsTo(... anchor=len(lines))` 不 panic。
  - [x] bottom line target Y 位于最后一行之后。
  - [x] `Timeline.ScrollToIndex == len(lines)` 不被 clamp 到最后一行。
  - [x] `ScrollMaxOffset` 使用 bottom line 后的 `curPos`。
  - [x] bottom line delayed / force spring 行为对齐 ref。
  - [x] bottom line Y spring 参数和普通歌词行同步。

- `lyrics/user_scroll_test.go`
  - [x] 用户滚动边界有效。
  - [x] seek 到歌词结束后，scroll target 可以是 bottom index。
  - [x] wheel scroll 会标记 `IsScrolled` 并 clamp 到边界。

- `lyrics/presentation_test.go`
  - [x] `SetHidePassedLines(true)` 会更新 layout 状态。
  - [x] interlude 期间 hide-passed 边界使用 `AnchorLineIndex + 1`。

- `lyrics/interlude_dots_test.go`
  - [x] dots 使用 ref 风格时间段动画。
  - [x] dots 退场在 interlude active 的最后阶段完成。
  - [x] dots 更新 metrics 时不重置 X/Y。

### 计划四实现状态

- [x] 4.1 新增 `BottomLine` 数据结构。
- [x] 4.2 `Lyrics` / `LyricsComponent` 持有并暴露 bottom line API。
- [x] 4.3 支持 `scrollToIndex == len(lines)` 作为 bottom line focus。
- [x] 4.4 bottom line 接入 `calcLayout` 末尾 transform。
- [x] 4.5 滚动边界包含 bottom line 之后的 `curPos`。
- [x] 4.6 bottom line spring 行为和参数同步对齐 ref。
- [x] 4.7 hidePassedLines API 与 presentation 语义对齐 ref。
- [x] 4.8 测试和文档收尾。

### 计划四第一版完成标准

- [x] `Lyrics` 持有 bottom line 状态。
- [x] `scrollToIndex == len(lines)` 被视为 bottom line focus。
- [x] `scrollLyricsTo` 支持 bottom index，不越界、不误加入 render set。
- [x] bottom line 参与末尾布局和滚动边界。
- [x] 空 bottom line 不绘制、不改变视觉内容。
- [x] `go test ./...` 通过。

### 计划四暂缓内容

- bottom line 文本/歌曲信息 UI 暂缓。
- bottom line 真实 blur 渲染暂缓。
- hide passed lines UI 或快捷键暂缓。
- 真实 blur shader 仍暂缓。

## 当前计划进度

为避免和早期总体路线中的“阶段 1～10”混淆，后续开发统一使用“计划一、计划二……”命名。

### 已完成

- [x] 计划一：timeline / optimize / presentation 基础状态。
- [x] 计划二：Spring 滚动、ref 风格布局、行间 delay、每帧 tick。
- [x] 计划三：完整 presentation、interlude dots、用户滚动、pause/resume、X 轴布局对齐。
- [x] 计划四：BottomLine、末尾滚动边界、hidePassedLines API/语义。
- [x] 计划五 A：真实 blur 渲染。
- [x] 计划五 C1：BottomLine 通用文本槽与 `setMusic` 创作者信息。

### 已完成补充

#### 计划五 A：真实 blur 渲染

状态：已完成。

目标：把已计算的 `BlurLevel` 接入 Ebiten 渲染层，尽量对齐 ref 的 `filter: blur(...)` 行为。

原则：

- [x] blur 最大值按 ref 风格 clamp，并通过 `BlurStrength` 支持运行时视觉调参。
- [x] active 行、用户滚动中、`EnableBlur=false` 时 blur 为 0。
- [x] 真实 blur 已接入所有需要模糊的歌词行，不再只限制 preview bitmap 行。
- [x] 使用 blur cache，避免每帧重复生成模糊图像。
- [x] blur 半径按 `lp.LP(...)` 转换，避免逻辑像素和屏幕像素比例导致效果过弱。
- [x] debug panel 已提供 `启用歌词模糊` 开关和 `歌词模糊强度` 滑块。

实现要点：

- `Line.BlurLevel` 由 presentation 写入，渲染层根据该值决定是否走 blur shader。
- `lyrics/layer_renderer.go` 使用 `filters.BlurImageShader` 绘制模糊后的行图像。
- `BlurImage` / `BlurCacheSource` / `BlurCacheKey` 缓存模糊结果，并在文本或渲染源变化时清理。
- `canUseStaticLayer()` 会排除带 blur 的行，避免静态层缓存吞掉动态模糊状态。
- `Lyrics.SetBlurStrength` / `LyricsComponent.SetBlurStrength` 用于调试和后续配置化。

#### 计划五 C1：BottomLine 通用文本槽与 `setMusic` 创作者信息

状态：已完成。

目标：在不破坏 ref 式“底部扩展槽”语义的前提下，让 bottom line 可以绘制开发者传入的通用文本，并先接入 WebSocket 的 `setMusic` 创作者信息。

实现要点：

- `BottomLine` 增加文本测量、文本图像缓存、blur cache、字体信息和绘制能力。
- `LyricsComponent.SetBottomLineText` / `ClearBottomLine` 继续作为通用开发者入口。
- `DrawLyricsDynamic` 绘制 bottom line，使其和 spring / focus / blur 状态一起动态更新。
- `Home` 订阅 `ws:setMusic`，从 `artists` 生成 `创作者：歌手A，歌手B` 并写入 bottom line。
- `Home.currentCreatorText` 持久保存当前创作者文本，解决 `setMusic -> setLyric` 顺序下歌词重建导致 bottom line 丢失的问题。
- BottomLine 左侧 padding 复用 `lineBasePadding`，和普通歌词文本左边对齐。

暂缓内容：

- 结构化两行歌曲信息 UI 暂缓，不在 C1 中实现。
- TTML metadata 自动填充暂缓。
- 封面、专辑、歌曲名等更完整底部卡片暂缓，避免偏离 ref 的底部扩展槽设计。

### 下一步候选与计划六

#### 计划五 B：视觉调参/回归

状态：并入计划六。

目标：不增加新功能，只调 Apple Music 观感。该目标已经从单独的“计划五 B”升级为“计划六：整体 Apple Music 观感回归与精修”。

可调项：

- Spring 参数和 per-line delay。
- `lineBasePadding` / duet avoidance。
- dots padding、margin、fade timing。
- pause/resume 时背景行 alpha。
- 用户滚动保持时间与边界。

#### 计划五 C：BottomLine 文本/歌曲信息 UI

目标：让 bottom line 真正绘制文本或歌曲信息。

当前状态：

- [x] C1：通用文本槽、真实绘制、`setMusic` 创作者信息接入。
- [ ] C2：结构化 `Title / Artist / Album` 两行 UI。
- [ ] C3：TTML metadata / WebSocket song info 的完整自动填充策略。

暂缓原因：

- ref 的 bottom line 主要提供 DOM 扩展点，当前 C1 已保留自由文本入口。
- C2/C3 需要进一步定义文本来源、样式、对齐和绘制缓存策略。

## 计划六：整体 Apple Music 观感回归与精修

计划六目标：不再优先新增架构能力，而是围绕 `ref/applemusic-like-lyrics/packages/core` 做一次端到端视觉回归，把已完成的 timeline、spring、layout、dots、blur、bottom line、用户滚动等能力调成一个稳定的 Apple Music 风格基线。

计划六不是早期路线里的“阶段 6”。早期“阶段 6：动态弹簧参数”已经完成，并归入计划二与计划四；当前“计划六”是新的后续迭代。

### 计划六 A：视觉回归清单与基线样例

状态：已完成，详见 `docs/plan-six-visual-regression-checklist.md`。

目标：先明确要观察什么，避免后续调参变成主观乱调。

- [x] 列出至少 5 类样例：普通逐词歌词、快速连唱、多热行/重叠、长间奏、带背景/对唱、歌曲末尾 bottom line。
- [x] 对照 ref 记录每类样例的预期行为：滚动时机、行透明度、缩放、blur、dots、末尾滚动。
- [x] 建立手动回归 checklist，记录每次调参前后的视觉差异。
- [x] 明确“可接受差异”：Ebiten 渲染与 DOM/CSS 不能完全一致，但高层行为要一致。

6A 产物：

- `docs/plan-six-visual-regression-checklist.md` 记录 ref 依据、A～G 七类回归样例、手动回归流程、可接受差异和完成标准。

### 计划六 B：滚动与 spring 观感精修

状态：已完成第一轮 ref 对齐。

目标：校准纵向滚动的物理感、延迟感和锚点，让快速歌词、seek、interlude、末尾 bottom focus 都稳定。

- [x] 复查 `computeLinePosYSpringParams` 与 ref 的 seeking / interlude / normal 参数切换。
- [x] 调整 per-line delay 与 delay 衰减，避免长列表滚动拖尾过重或过硬。
- [x] 校准 `AlignPosition` 与 `LayoutAlignAnchorCenter` 的实际视觉位置。
- [x] 回归 `scrollToIndex == len(lines)` 时 bottom line focus 的位置和末尾留白。
- [x] 确认用户滚动后 5 秒 reset 与 ref 行为一致，且 reset 不产生跳变。

实现要点：

- Y spring 默认参数对齐 ref：`mass: 0.9, damping: 15, stiffness: 90`。
- 背景行 scale spring 对齐 ref：`mass: 1, damping: 20, stiffness: 50`。
- 动态 Y spring 保持 ref 公式：普通播放使用 `stiffness: 170..220`，`damping = sqrt(stiffness) * 2.2`，并保留 `mass: 0.9`。
- 初始布局与 ref 的 sync layout 对齐：初始布局不叠加 per-line delay。
- per-line delay 只随非背景行推进，和 ref 中 `if (!line.isBG) delay += baseDelay` 一致。
- 歌曲末尾只有 bottom line 有内容时才使用 `scrollToIndex == len(lines)`；空 bottom line 时回到最后一行，避免无内容底栏抢焦点。
- 保持默认 `AlignPosition = 0.35` 与 `LayoutAlignAnchorCenter`，未在第一轮中改动视觉锚点常量。

### 计划六 C：行 presentation、blur 与背景行精修

状态：已完成第一轮 ref 对齐。

目标：统一 active / buffered / inactive / passed / background 行的 alpha、scale、blur 组合，减少边缘状态跳变。

- [x] 复查 `computeLinePresentation` 与 ref 的 active、buffered、non-dynamic、background、hidePassedLines 规则。
- [x] 校准 inactive alpha、background alpha、active scale、background scale。
- [x] 检查 blur 强度默认值，确认不依赖 debug slider 才能得到合理观感。
- [x] 回归用户滚动中 blur 关闭、active 行 blur 为 0、bottom focused blur 为 0。
- [x] 检查 pause/resume 时背景歌词显示是否稳定。

实现要点：

- `newLayoutState()` 默认启用 blur，`BlurStrength` 回到 ref 基线倍率 `1`。
- `Home.OnCreate()` 默认启用歌词 blur，并把 debug slider 初始值设为 `1`。
- 背景行 active 与暂停显示 alpha 对齐 ref CSS 的 `0.4`，不再使用主行式 `1 / 0.85`。
- 背景行播放中 inactive 仍不占位、不显示；active 或暂停时 reserve space。
- `applyLinePresentation` 会把 presentation 的 render mode 写回 line，避免后续渲染忽略 active / inactive 模式切换。
- 保留 blur 运行时调试倍率，但默认值为 ref 的 `1`，用于后续视觉微调而非默认补偿。

### 计划六 D：interlude dots 精修

状态：已完成第一轮 ref 对齐。

目标：让长间奏 dots 在时机、位置、大小和透明度上更接近 ref。

- [x] 回归 interlude gap 计算：最小间隔、开始时间、结束时间、`nextLine.startTime - 250ms`。
- [x] 校准 dots fade in、enter scale、last 750ms exit、last 375ms fade out。
- [x] 调整 dots padding、margin、dot size、gap，使其和 ref CSS 行间距一致。
- [x] 检查 dots 与下一行 duet 方向对齐。
- [x] 确认 interlude 结束后 dots 不残留、不跳到左上角。

实现要点：

- 保持 interlude 时间逻辑：当前时间偏移 `+20ms`，gap 结束为下一行开始前 `250ms`，最小 gap `4000ms`。
- 保持 dots 动画时序：前 `500ms` 透明、`500ms~1000ms` fade in、前 `2000ms` scale enter、最后 `750ms` exit scale、最后 `375ms` fade out。
- 垂直 padding 改为 ref CSS 语义：`padding-block: 2.5%` 基于 dots 自身尺寸，而不是视口高度。
- dots X 轴对齐回到 ref 行为：普通 dots `left = 0`，duet dots `left = playerWidth - dotsWidth`。
- `InterludeDots` 保存 `IsDuet` 状态，绘制时 duet 以右边缘作为缩放锚点，模拟 ref 的 `.interludeDots.duet { right: 0; transform-origin: center; }` 视觉效果。
- `SetInterlude(nil)` 会清空 Active、时间、alpha/scale 和 duet 状态。

### 计划六 E：BottomLine 回归与轻量扩展准备

状态：已完成第一轮 ref 对齐。

目标：把 C1 的通用文本槽纳入整体回归，并为后续 C2/C3 留好接口但不急着实现卡片式 UI。

- [x] 回归 `setMusic -> setLyric` 与 `setLyric -> setMusic` 两种顺序。
- [x] 检查 bottom line 文本左对齐、focus alpha、blur、spring 位置。
- [x] 检查无 artist 时 `ClearBottomLine` 不影响末尾滚动安全。
- [x] 检查字体切换、窗口 resize 后 bottom line 文本缓存和高度更新。
- [x] 评估是否需要新增 `SetBottomLineInfo`，但计划六内默认不实现结构化 UI。

实现要点：

- 参考 ref 的 `.bottomLine:empty` 与 `innerHTML.trim().length > 0` 语义，BottomLine 现在用 `HasContent()` 判断可见内容；纯空白文本会被视为空 slot，高度为 `0`，歌曲结束后不会抢占 bottom focus。
- 时间线末尾目标从直接读取 `Bottom.Active` 改为 `Bottom.HasContent()`，和 ref 的 `hasBottomContent` 一致。
- `SetFontSize` 会同步调用 `Bottom.SetFont(...)`，字体大小热切换时 bottom line 的字号、padding、测量高度和图像缓存一起更新。
- 保持 C1 的通用文本槽定位：不在计划六中新增结构化 `SetBottomLineInfo` 或卡片式 UI，后续仍归入计划五 C2/C3。
- 新增回归测试覆盖空白 bottom line、末尾 target、用户滚动中 bottom blur 为 0，并保留已有 resize / font / spring / focus 测试。

### 计划六 F：性能、缓存与测试收口

目标：保证视觉精修不会破坏性能和状态稳定性。

- [x] 检查 blur cache、bottom line image cache、static/dynamic layer 命中情况。
- [x] 回归快速切歌、频繁 `setLyric`、频繁 `setMusic`、字体配置热切换。
- [x] 为新发现的边界补测试，优先测状态和缓存，不做脆弱的像素级测试。
- [x] 每轮调参后运行 `go test ./lyrics`、`go test ./pages`、`go test ./...`。
- [x] 在文档中记录最终推荐参数和仍可接受的差异。

实现要点：

- 新增 `SharedImageCacheStats()` 与 `Lyrics.RenderCacheStats()`，用于检查 shared text mask / gradient 引用数、可见行 bitmap、dirty bitmap、line blur、bottom line image / blur cache 状态。
- 新增缓存回归测试覆盖 shared image 引用计数、释放归零、line / bottom cache 统计，以及 static layer signature 的稳定命中与 transform 变更失效。
- 快速切歌路径仍由 `LyricsComponent.SetLyrics()` 先 dispose 旧歌词，再 `PurgeSharedImageCache()`，并保留 crossfade snapshot；频繁 `setMusic` 只更新 bottom line 文本并触发 relayout，不重建整首歌词。
- 字体热切换继续通过 `SetFontSize` / `SetFont` 重排歌词、bottom line 与 static layer signature，不改变计划六 B～E 的 ref 基线参数。
- 可接受差异保持 6A 结论：不做像素级缓存测试，重点保证状态、引用计数和静态层失效条件稳定。

### 命名约定

- “阶段 1～10”只表示早期总体路线，不再作为当前迭代编号。
- 当前迭代使用“计划一、计划二……计划六”。
- 早期“阶段 6：动态弹簧参数”已经完成，不等于当前“计划六”。
- 如果继续推进结构化 bottom line，应称为“计划五 C2 / C3”，不要称为“阶段 5”。
