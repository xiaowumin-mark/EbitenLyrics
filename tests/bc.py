import numpy as np
import matplotlib.pyplot as plt
from matplotlib.widgets import Button

# 初始控制点
points = np.array([[0, 0], [1, 2], [3, 3], [4, 0]])

def bezier_curve(P, n=100):
    t = np.linspace(0, 1, n)
    # 使用矩阵形式计算贝塞尔曲线
    B = np.array([
        (1-t)**3,
        3*(1-t)**2*t,
        3*(1-t)*t**2,
        t**3
    ]).T
    return B @ P

fig, ax = plt.subplots()
ax.set_title("三次贝塞尔曲线交互演示")
ax.set_xlim(-1, 5)
ax.set_ylim(-1, 4)

# 绘制初始曲线和控制点
curve_line, = ax.plot([], [], 'b-', lw=2)
control_line, = ax.plot(points[:,0], points[:,1], 'r--')
control_points, = ax.plot(points[:,0], points[:,1], 'ro')

def update_curve():
    curve = bezier_curve(points)
    curve_line.set_data(curve[:,0], curve[:,1])
    control_line.set_data(points[:,0], points[:,1])
    control_points.set_data(points[:,0], points[:,1])
    fig.canvas.draw_idle()

update_curve()

# 鼠标拖动事件
dragging_point = None
def on_press(event):
    global dragging_point
    if event.inaxes != ax: return
    for i, (x, y) in enumerate(points):
        if abs(event.xdata - x) < 0.2 and abs(event.ydata - y) < 0.2:
            dragging_point = i

def on_release(event):
    global dragging_point
    dragging_point = None

def on_motion(event):
    global dragging_point
    if dragging_point is None: return
    points[dragging_point] = [event.xdata, event.ydata]
    update_curve()

fig.canvas.mpl_connect('button_press_event', on_press)
fig.canvas.mpl_connect('button_release_event', on_release)
fig.canvas.mpl_connect('motion_notify_event', on_motion)

plt.show()
