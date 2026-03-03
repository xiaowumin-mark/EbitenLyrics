package bgrender

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
