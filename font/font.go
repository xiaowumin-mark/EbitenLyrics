package font

// 文件说明：字体对象的轻量封装。
// 主要职责：按字号从字体源创建 Ebiten 文本绘制所需的 `text.Face`。

import (
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func GetFace(font *text.GoTextFaceSource, size float64) text.Face {
	if font == nil || size <= 0 {
		return nil
	}
	return &text.GoTextFace{
		Source: font,
		Size:   size,
	}
}
