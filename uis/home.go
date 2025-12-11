package uis

import (
	ft "EbitenLyrics/font"
	"image/color"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/colornames"
)

// 定义一些从截图取色的常量
var (
	ColorBgDark     = hexToColor("050b14") // 最深背景
	ColorBgMedium   = hexToColor("0d1326") // 内容区背景
	ColorBgLight    = hexToColor("161f36") // 顶栏/底栏背景
	ColorAccentBlue = hexToColor("1d2b4e") // 按钮/卡片背景
	ColorText       = colornames.White
	ColorTextGray   = colornames.Gray
)

// --- UI 构建核心逻辑 ---

func CreateUI(font *text.GoTextFaceSource) *ebitenui.UI {
	// 根容器：使用 RowLayout 垂直排列 (TopBar -> Content -> PlayerBar)
	rootContainer := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(ColorBgDark)),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(0),
		)),
	)

	// --- A. 顶部栏 (Header) ---
	header := createHeader(font)
	rootContainer.AddChild(header)

	// --- B. 中间内容区 (Content) ---
	// 设置 StretchVertical: true 让它占据剩余空间
	content := createContent(font)
	rootContainer.AddChild(content)

	// --- C. 底部播放栏 (Footer) ---
	footer := createFooter(font)
	rootContainer.AddChild(footer)

	return &ebitenui.UI{
		Container: rootContainer,
	}
}

// 创建顶部栏
func createHeader(font *text.GoTextFaceSource) *widget.Container {
	c := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(ColorBgDark)),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(15)),
			widget.RowLayoutOpts.Spacing(15),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Stretch:   true,
				MaxHeight: 60, // 固定高度
			}),
		),
	)
	f := ft.GetFace(font, 24)
	// 1. Logo 文本
	c.AddChild(widget.NewText(
		widget.TextOpts.Text("AMLL Player", &f, ColorText),
	))

	// 2. Tags (胶囊标签)
	//c.AddChild(createTag("有可用更新", font, hexToColor("2a3b55")))
	//c.AddChild(createTag("SMTC 监听模式", font, hexToColor("0f3d2e")))

	// 3. 占位符 (Spacer)，把后面的按钮挤到右边
	// 这里的技巧是创建一个空的 Container，设为 StretchHorizontal
	spacer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(widget.RowLayoutOpts.Direction(widget.DirectionHorizontal))),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{
			//StretchHorizontal: true,
			Stretch: true,

			Position: widget.RowLayoutPositionEnd,
		})),
	)
	c.AddChild(spacer)

	// 4. 右侧按钮组
	c.AddChild(createButton("Q", font, ColorAccentBlue)) // 搜索图标代替
	c.AddChild(createButton("+ 新建播放列表", font, ColorAccentBlue))
	c.AddChild(createButton("=", font, ColorAccentBlue)) // 菜单图标代替

	return c
}

// 创建中间内容区
func createContent(font *text.GoTextFaceSource) *widget.Container {
	// 外层容器，深色背景
	contentArea := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(ColorBgMedium)),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout(
			widget.AnchorLayoutOpts.Padding(widget.NewInsetsSimple(30)),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				//StretchHorizontal: true,
				//StretchVertical:   true, // 关键：填充剩余高度
				Stretch: true,
			}),
		),
	)

	// 创建截图中的 "Music" 播放列表卡片
	card := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(ColorAccentBlue)), // 卡片背景
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(15)),
			widget.RowLayoutOpts.Spacing(20),
		)),
		// 在 AnchorLayout 中定位在顶部
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionStart,
				VerticalPosition:   widget.AnchorLayoutPositionStart,
				StretchHorizontal:  true, // 让卡片横向拉伸
			}),
			widget.WidgetOpts.MinSize(0, 100),
		),
	)

	// 卡片左侧：封面图 (模拟)
	coverArt := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(colornames.Forestgreen)),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.MinSize(80, 80)),
	)
	card.AddChild(coverArt)

	// 卡片右侧：文本信息 (垂直排列)
	infoCol := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(5),
		)),
	)
	f := ft.GetFace(font, 22)
	infoCol.AddChild(widget.NewText(
		widget.TextOpts.Text("music", &f, ColorText),
	))
	f = ft.GetFace(font, 14)
	infoCol.AddChild(widget.NewText(
		widget.TextOpts.Text("71 首歌曲 - 创建于 2025/9/13", &f, ColorTextGray),
	))

	card.AddChild(infoCol)
	contentArea.AddChild(card)

	return contentArea
}

// 创建底部播放栏
func createFooter(font *text.GoTextFaceSource) *widget.Container {
	footer := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(ColorBgDark)),
		// 使用 AnchorLayout 以便让播放控件完美居中
		widget.ContainerOpts.Layout(widget.NewAnchorLayout(
			widget.AnchorLayoutOpts.Padding(widget.NewInsetsSimple(10)),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Stretch:   true,
				MaxHeight: 90, // 固定高度

			}),
			widget.WidgetOpts.MinSize(0, 90),
		),
	)

	// 1. 左侧：正在播放的歌曲信息
	leftInfo := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(10),
		)),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionStart,
			VerticalPosition:   widget.AnchorLayoutPositionCenter,
		})),
	)
	// 封面
	leftInfo.AddChild(widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(colornames.Purple)),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.MinSize(60, 60)),
	))
	f := ft.GetFace(font, 16)
	// 歌名文字
	textCol := widget.NewContainer(widget.ContainerOpts.Layout(widget.NewRowLayout(widget.RowLayoutOpts.Direction(widget.DirectionVertical))))
	textCol.AddChild(widget.NewText(widget.TextOpts.Text("Zoo (From \"Zootopia\")", &f, ColorText)))
	f = ft.GetFace(font, 12)
	textCol.AddChild(widget.NewText(widget.TextOpts.Text("Disney/Shakira", &f, ColorTextGray)))
	leftInfo.AddChild(textCol)

	footer.AddChild(leftInfo)

	// 2. 中间：播放控制按钮 (上一曲，暂停，下一曲)
	controls := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(20),
		)),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionCenter, // 绝对居中
			VerticalPosition:   widget.AnchorLayoutPositionCenter,
		})),
	)
	controls.AddChild(createButton("<<", font, ColorBgDark))                        // 上一曲
	controls.AddChild(createButton("||", font, colornames.White, colornames.Black)) // 播放/暂停 (高亮)
	controls.AddChild(createButton(">>", font, ColorBgDark))                        // 下一曲
	footer.AddChild(controls)

	// 3. 右侧：菜单按钮
	rightMenu := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(widget.RowLayoutOpts.Direction(widget.DirectionHorizontal))),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionEnd,
			VerticalPosition:   widget.AnchorLayoutPositionCenter,
		})),
	)
	rightMenu.AddChild(createButton("List", font, ColorAccentBlue))
	footer.AddChild(rightMenu)

	return footer
}

// --- 辅助函数 ---

// 创建一个带背景色的简单按钮
func createButton(label string, font *text.GoTextFaceSource, bgColor color.Color, textColors ...color.Color) *widget.Button {
	txtColor := colornames.White
	if len(textColors) > 0 {
		txtColor = textColors[0].(color.RGBA)
	}
	f := ft.GetFace(font, 14)
	return widget.NewButton(
		widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.MinSize(40, 40)),
		widget.ButtonOpts.Image(&widget.ButtonImage{
			Idle:    image.NewNineSliceColor(bgColor),
			Hover:   image.NewNineSliceColor(Mix(bgColor, colornames.White, 0.2)),
			Pressed: image.NewNineSliceColor(Mix(bgColor, colornames.Black, 0.2)),
		}),

		widget.ButtonOpts.Text(label, &f, &widget.ButtonTextColor{Idle: txtColor}),
		widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(10)),
	)
}

// 创建类似胶囊的标签
func createTag(label string, font *text.GoTextFaceSource, bgColor color.Color) *widget.Container {
	c := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(bgColor)),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Padding(&widget.Insets{Top: 5, Bottom: 5, Left: 10, Right: 10}),
		)),
	)
	f := ft.GetFace(font, 12)
	c.AddChild(widget.NewText(widget.TextOpts.Text(label, &f, ColorAccentBlue))) // 这里的字色可能要调整
	// 简单起见，这里文字用白色
	return widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(bgColor)),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout(widget.AnchorLayoutOpts.Padding(&widget.Insets{Top: 5, Bottom: 5, Left: 10, Right: 10}))),
		widget.ContainerOpts.AutoDisableChildren(),                                            // 标签通常不可点击
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{})), // Row布局数据
	)
	// 上面的Tag实现稍微复杂，简化直接返回Button如果不交互：
	/*return widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(bgColor)),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Padding(widget.Insets{Top: 5, Bottom: 5, Left: 10, Right: 10}),
		)),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.MinSize(0, 25)),
	)*/
}

// 颜色混合辅助函数 (用于做Hover效果)
func Mix(c1, c2 color.Color, ratio float64) color.Color {
	r1, g1, b1, _ := c1.RGBA()
	r2, g2, b2, _ := c2.RGBA()
	return color.RGBA{
		R: uint8((float64(r1)*(1-ratio) + float64(r2)*ratio) / 257),
		G: uint8((float64(g1)*(1-ratio) + float64(g2)*ratio) / 257),
		B: uint8((float64(b1)*(1-ratio) + float64(b2)*ratio) / 257),
		A: 255,
	}
}

// Hex转Color辅助函数
func hexToColor(s string) color.Color {
	// 简单解析，假设输入如 "050b14"
	type hex struct {
		val uint8
	}
	// ... 省略复杂的Hex解析，这里直接返回近似色以便演示运行
	// 实际代码中请使用标准的 Hex 解析库
	if s == "050b14" {
		return color.RGBA{5, 11, 20, 255}
	}
	if s == "0d1326" {
		return color.RGBA{13, 19, 38, 255}
	}
	if s == "161f36" {
		return color.RGBA{22, 31, 54, 255}
	}
	if s == "1d2b4e" {
		return color.RGBA{29, 43, 78, 255}
	}
	return colornames.Black
}
