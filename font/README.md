# `font` Package

中文 / English README for the passive, request-driven font manager used by this project.

## Overview / 概览

### 中文

`font` 包是一个面向 `Ebiten` `text/v2` 的字体解析与回退工具包，特点是：

- 被动请求式，不在启动时主动全量初始化字体
- 通过 `FontRequest` 描述字体需求，而不是让业务层长期持有字体句柄
- 支持多字体族请求、有序 fallback、自定义 fallback 规则
- 支持按字符的精准回退
- 使用 `mmap` 加载字体文件，避免 `ReadFile`
- 使用 LRU 管理字体源，减少长时间运行时的内存压力
- 支持 `.ttc` / collection font，并按字重与斜体挑选子字体
- 当前对 Windows 做了更优的系统字体发现逻辑

这个包的目标是：

- 让业务层只关心“我要什么字体”
- 让字体解析、系统字体发现、fallback、内存生命周期都集中在 `font` 包内部

### English

The `font` package is a passive, request-driven font manager for `Ebiten` `text/v2`.

Key properties:

- Passive resolution instead of eager startup scanning
- `FontRequest` describes font intent; business code does not keep raw file handles
- Ordered multi-family requests and custom fallback rules
- Per-rune fallback for missing glyphs
- `mmap`-based font loading, never `ReadFile`
- LRU-managed font sources for long-running sessions
- `.ttc` / collection font support with weight + italic selection
- Windows gets a faster system-font discovery path than generic directory walking

The design goal is to keep application code focused on intent while the `font` package handles discovery, fallback, loading, and memory management internally.

## Main Concepts / 核心概念

### `FontRequest`

### 中文

`FontRequest` 是字体请求对象：

```go
type FontRequest struct {
    Families []string
    Weight   Weight
    Italic   bool
}
```

含义：

- `Families`: 有序字体族列表，越靠前优先级越高
- `Weight`: 目标字重，例如 `WeightRegular` / `WeightMedium` / `WeightBold`
- `Italic`: 是否倾向斜体

例子：

```go
req := font.FontRequest{
    Families: []string{"Inter", "Segoe UI", "Microsoft YaHei"},
    Weight:   font.WeightMedium,
    Italic:   false,
}
```

### English

`FontRequest` is the primary description of font intent:

```go
type FontRequest struct {
    Families []string
    Weight   Weight
    Italic   bool
}
```

Meaning:

- `Families`: ordered family list, highest priority first
- `Weight`: desired weight, such as `WeightRegular`, `WeightMedium`, or `WeightBold`
- `Italic`: whether italic faces are preferred

Example:

```go
req := font.FontRequest{
    Families: []string{"Inter", "Segoe UI", "Microsoft YaHei"},
    Weight:   font.WeightMedium,
    Italic:   false,
}
```

### Passive Resolution / 被动解析

### 中文

`font` 包不会在启动时预扫全部字体并提前构建所有 face。

只有在调用这些 API 时才会工作：

- `ResolveChain(req)`
- `GetFace(req, size)`
- `GetFaceForText(req, size, content)`
- `FindFaceForRune(r, chain)`

其中：

- `ResolveChain` 只做描述级解析，不会把整条链全部强制加载成 `GoTextFaceSource`
- `GetFace` 会优先拿主字体
- `GetFaceForText` 会按文本内容检查是否需要加载 fallback
- `FindFaceForRune` 用于基于单个 rune 的精准回退

### English

The package does not eagerly prebuild every font or face at startup.

Work only happens when one of these APIs is called:

- `ResolveChain(req)`
- `GetFace(req, size)`
- `GetFaceForText(req, size, content)`
- `FindFaceForRune(r, chain)`

Behavior:

- `ResolveChain` resolves descriptors only
- `GetFace` primarily loads the primary face path
- `GetFaceForText` loads additional fallbacks only when the text actually needs them
- `FindFaceForRune` is the per-rune fallback API

## Public API / 对外 API

### Construction / 创建

```go
manager := font.NewFontManager(16)
```

### 中文

`16` 是字体源 LRU 大小，表示最多缓存多少个已 mmap 并解析完成的字体文件。

如果你不想自己管理实例，也可以直接使用默认单例：

```go
manager := font.DefaultManager()
```

### English

The `16` above is the LRU size for loaded font files.

If you prefer a shared singleton:

```go
manager := font.DefaultManager()
```

### Default Request / 默认请求

```go
req := font.DefaultRequest()
families := font.DefaultFamilies()
```

### 中文

默认字体会根据平台给出一组安全 fallback 链。

### English

Default families are platform-aware and provide a safe built-in fallback chain.

### Custom Fallback Rules / 自定义回退规则

```go
manager.RegisterFallback(map[string][]string{
    "Inter":   {"Segoe UI", "Microsoft YaHei"},
    "SF Pro":  {"SF Pro Text", "SF Pro Display"},
})
```

或使用默认单例：

```go
font.RegisterFallback(map[string][]string{
    "Inter": {"Segoe UI", "Microsoft YaHei"},
})
```

### 中文

规则会插入到请求链中，优先级在：

1. 用户显式请求的 `Families`
2. 用户注册的 fallback 规则
3. 系统内置安全 fallback
4. last resort 字体

### English

Fallback rules are inserted between:

1. user-requested families
2. user-registered fallback rules
3. built-in system fallbacks
4. the last-resort font

### Register Custom Font Files / 注册自定义字体文件

```go
err := manager.RegisterCustomFontPath("My Brand Sans", `D:\fonts\BrandSans-Regular.ttf`)
```

之后即可直接请求：

```go
req := font.FontRequest{
    Families: []string{"My Brand Sans", "Segoe UI"},
    Weight:   font.WeightRegular,
}
```

### 中文

这适合：

- 项目内置私有字体
- 系统未安装但你知道路径的字体
- 想通过别名统一请求的字体

### English

This is useful for:

- bundled private fonts
- fonts not installed system-wide
- stable aliases for project-specific fonts

### Resolve a Chain / 解析字体链

```go
chain, err := manager.ResolveChain(req)
```

返回值包含：

- `Primary`
- `Fallbacks`
- `Families`

### 中文

这一步适合调试和日志，不一定需要在业务中频繁调用。

### English

This API is useful for debugging and logging. Most rendering code can go straight to `GetFace` or `GetFaceForText`.

### Get a Face / 获取 face

```go
face, err := manager.GetFace(req, 48)
```

### 中文

适用于：

- 已知主要使用主字体
- 不需要为具体文本做 fallback 预判

### English

Use this when the primary face is enough or when you do not want content-aware fallback selection.

### Get a Face for Specific Text / 按文本获取 face

```go
face, err := manager.GetFaceForText(req, 48, "Hello 世界")
```

### 中文

这是推荐的渲染入口。  
它会根据文本内容决定是否把 fallback 追加进最终的 `MultiFace`。

### English

This is the recommended rendering API.  
It builds a face chain based on actual text content and only loads additional fallback faces when needed.

### Find a Face for One Rune / 按单个字符找 face

```go
face, err := manager.FindFaceForRune('你', []string{"Inter", "Segoe UI", "Microsoft YaHei"})
```

### 中文

适用于：

- 业务层自己做逐字符渲染
- 需要显式控制每个 rune 的 fallback

### English

Useful for:

- custom per-rune rendering
- explicit rune-level fallback handling

### Parse from Config / 从配置读取请求

```go
req, err := manager.LoadRequestFromFile("config/font.json", font.DefaultRequest())
```

也可以：

```go
req, err := manager.ParseRequest(base, cfgMap)
req, err := manager.ApplyEnvRequest(base)
```

支持的字段：

- `family`
- `families`
- `weight`
- `italic`
- `path`
- `fontPath`
- `font_file`
- `file`

示例：

```json
{
  "font": {
    "families": ["SF Pro", "Segoe UI", "Microsoft YaHei"],
    "weight": 500,
    "italic": false
  }
}
```

或直接绑定单文件字体：

```json
{
  "font": {
    "path": "./assets/fonts/BrandSans-Regular.ttf",
    "weight": 400
  }
}
```

### Environment Variables / 环境变量

支持：

- `EBITENLYRICS_FONT_FAMILY`
- `EBITENLYRICS_FONT_WEIGHT`
- `EBITENLYRICS_FONT_ITALIC`
- `EBITENLYRICS_FONT_PATH`

## Fallback Resolution Order / 回退链顺序

### 中文

最终链路按这个顺序拼接：

1. `FontRequest.Families`
2. `RegisterFallback` 为这些 family 配置的用户规则
3. 平台内置系统 fallback
4. last-resort 字体

举例：

```go
req := font.FontRequest{
    Families: []string{"Inter"},
    Weight:   font.WeightMedium,
}

manager.RegisterFallback(map[string][]string{
    "Inter": {"Segoe UI", "Microsoft YaHei"},
})
```

在 Windows 上最终可能变成：

```text
Inter
Segoe UI
Microsoft YaHei
Arial
Segoe UI Symbol
...
last resort
```

### English

The effective fallback chain is built in this order:

1. `FontRequest.Families`
2. user rules from `RegisterFallback`
3. built-in platform fallback families
4. the last-resort font

## Platform Behavior / 平台行为

### Windows

### 中文

Windows 当前会优先走系统字体注册表：

- `HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Fonts`
- `HKCU\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Fonts`

这比递归扫描整个字体目录更快，也更接近系统真实安装字体列表。  
解析出字体路径后，`font` 包仍然会读取字体文件的真实 family/style/weight 信息，以确保匹配准确。

### English

On Windows, the package first reads the system font registry instead of recursively walking font directories.

Registry sources:

- `HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Fonts`
- `HKCU\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Fonts`

This is faster than a full directory walk and still preserves correctness because actual family/style/weight metadata is read from the font file itself.

### Other Platforms

### 中文

macOS / Linux / 其他平台暂时仍然走目录扫描降级路径。  
对外 API 不变，后续可以继续为不同平台接更原生的系统字体索引能力。

### English

Non-Windows platforms currently fall back to directory walking.  
The public API remains unchanged, so platform-specific improvements can be added later without changing call sites.

## Performance Notes / 性能说明

### 中文

当前包内做了这些优化：

- `mmap` 挂载字体文件
- LRU 管理 `GoTextFaceSource`
- 请求链缓存
- 文本内容到 fallback 结果的缓存
- glyph 覆盖检查缓存
- Windows 下使用注册表做系统字体发现

仍需注意：

- 第一次请求一个从未见过的 family，仍然可能有冷启动成本
- 混合语言文本会触发更多 fallback 检查
- 真实渲染性能不只取决于 `font` 包，也取决于 `text.Measure`、文本贴图生成、模糊和 shader

### English

Current optimizations include:

- `mmap`-based font loading
- LRU-managed `GoTextFaceSource`
- request-chain caching
- content-to-selected-font caching
- glyph-coverage caching
- Windows registry-backed system font discovery

Still important:

- the first request for a new family can still have cold-start cost
- mixed-language content may require more fallback checks
- total rendering cost also depends on `text.Measure`, text mask generation, blur, and shader work

## Memory Model / 内存模型

### 中文

字体文件通过 `mmap` 映射，不会使用 `os.ReadFile` 把整个文件复制进 Go heap。  
但 `mmap` 仍然会体现在进程工作集里，所以任务管理器看到的内存不等于 Go heap。

`FontManager.Stats()` 可以看到：

- `LoadedFiles`
- `MappedBytes`

### English

Font files are memory-mapped instead of being read into heap memory with `ReadFile`.  
Mapped memory still appears in process working set numbers, so Task Manager memory is not the same as Go heap usage.

`FontManager.Stats()` reports:

- `LoadedFiles`
- `MappedBytes`

## TTC / Font Collection Support / TTC 与字库集合支持

### 中文

`.ttc` 和其他 collection font 会被识别为多子字体文件。  
`font` 包会读取 collection 中的多个 face，并结合：

- family
- weight
- italic

来选择最合适的子字体索引。

### English

`.ttc` and other collection fonts are supported.  
The package inspects faces inside the collection and chooses the most suitable entry based on:

- family
- weight
- italic

## Recommended Usage / 推荐用法

### 中文

推荐业务层只保存：

- `*font.FontManager`
- `font.FontRequest`

不要长期保存：

- 字体文件句柄
- `*text.GoTextFaceSource`
- 人工维护的 fallback face 列表

推荐：

```go
manager := font.DefaultManager()
req := font.FontRequest{
    Families: []string{"SF Pro", "Segoe UI", "Microsoft YaHei"},
    Weight:   font.WeightMedium,
}

face, err := manager.GetFaceForText(req, 48, "Hello 世界")
if err != nil {
    return err
}
```

### English

Application code should usually keep only:

- `*font.FontManager`
- `font.FontRequest`

Avoid keeping:

- raw file handles
- long-lived `*text.GoTextFaceSource`
- manually managed fallback face lists

Recommended pattern:

```go
manager := font.DefaultManager()
req := font.FontRequest{
    Families: []string{"SF Pro", "Segoe UI", "Microsoft YaHei"},
    Weight:   font.WeightMedium,
}

face, err := manager.GetFaceForText(req, 48, "Hello 世界")
if err != nil {
    return err
}
```

## Debugging Tips / 调试建议

### 中文

如果你怀疑字体没有正确命中，优先检查：

- 请求的 family 名是否真实存在
- 是否命中了别名或 prefix family
- 配置里是否传入了错误路径
- 是否确实需要某个 CJK / emoji fallback

可以先调用：

```go
chain, err := manager.ResolveChain(req)
```

查看：

- `chain.Primary.Family`
- `chain.Primary.Style`
- `chain.Primary.Path`
- `chain.Fallbacks`

### English

If font selection looks wrong, first check:

- whether the requested family name is real
- whether a prefix or alias match was used
- whether the config points to the correct file
- whether the content actually needs extra CJK / emoji fallback

Use:

```go
chain, err := manager.ResolveChain(req)
```

Then inspect:

- `chain.Primary.Family`
- `chain.Primary.Style`
- `chain.Primary.Path`
- `chain.Fallbacks`

## Compatibility Promise / 兼容性说明

### 中文

当前优先保证：

- 对外 API 名称尽量稳定
- 业务层以 `FontRequest` 为中心的用法不变
- 平台优化尽量封装在 `font` 包内部

后续可能继续新增：

- Windows 持久化 family 索引缓存
- macOS CoreText 字体索引
- Linux Fontconfig 字体索引
- 更细粒度的 face / layout benchmark

### English

Current priorities:

- keep public API names stable
- keep `FontRequest`-driven usage stable
- encapsulate platform-specific optimization inside the `font` package

Possible future work:

- persistent Windows family index cache
- CoreText-backed macOS indexing
- Fontconfig-backed Linux indexing
- more detailed face and layout benchmarks
