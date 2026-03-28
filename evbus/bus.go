package evbus

// 文件说明：暴露全局事件总线实例。
// 主要职责：为页面、歌词组件和 WebSocket 数据流提供松耦合通信入口。

import "github.com/asaskevich/EventBus"

var Bus = EventBus.New()
