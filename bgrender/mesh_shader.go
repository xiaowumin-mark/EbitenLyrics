package bgrender

// 文件说明：网格渲染相关着色器加载与参数组织。
// 主要职责：把 CPU 侧网格数据传入 GPU 着色流程。

import (
	_ "embed"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

var (
	//go:embed shaders/mesh_bg.kage
	meshBGShaderSource []byte

	meshBGShaderOnce sync.Once
	meshBGShader     *ebiten.Shader
	meshBGShaderErr  error
)

func loadMeshBGShader() (*ebiten.Shader, error) {
	meshBGShaderOnce.Do(func() {
		meshBGShader, meshBGShaderErr = ebiten.NewShader(meshBGShaderSource)
	})
	return meshBGShader, meshBGShaderErr
}
