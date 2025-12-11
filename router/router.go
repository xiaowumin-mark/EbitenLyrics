package router

var scenes = map[string]Scene{}
var current Scene
var currentName string
var created = map[string]bool{}
var needFirstResize bool

func Add(name string, s Scene) {
	scenes[name] = s
}
func Go(name string, params map[string]any) {
	if current != nil {
		current.OnLeave()
	}

	s := scenes[name]

	if !created[name] {
		s.OnCreate()
		created[name] = true
	}

	s.OnEnter(params)

	current = s
	currentName = name

	// ⭐ 页面切换后下次 Update 会触发一次 isFirst
	needFirstResize = true
}

func Current() Scene {
	return current
}

func NeedFirstResize() bool {
	return needFirstResize
}

func ClearFirstResizeFlag() {
	needFirstResize = false
}
