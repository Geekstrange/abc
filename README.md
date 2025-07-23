# ABC (ADB Basic Controller)

## 工具概述

ABC (ADB Basic Controller) 是一款轻量级的 Android 设备命令行控制工具，通过原生 ADB 协议实现，无需依赖图形界面即可完成复杂操作。其核心设计聚焦于 零延迟响应 与 跨平台稳定性 (支持 Windows/Linux/macOS)

## 核心功能详解

### 1. 精准屏幕映射

- **比例自适应预览**：终端内动态生成与设备实际尺寸等比例的预览框，支持 16:9、18:9、21:9 等主流屏幕比例，解决传统终端控制中画面拉伸变形的问题。
- **实时指针定位**：通过 "●" 符号直观显示当前操作位置，坐标实时同步设备真实像素点，移动精度可通过源码`moveStep`参数自定义（默认 15 像素 / 步）。
- **网格辅助线**：可配置的网格线系统（默认间隔 5 字符），提升定位精度，支持自定义网格字符与间隔密度。

### 2. 灵活操作体系

- 基础交互

  ：

  - 方向键：控制指针上下左右移动
  - Enter 键：模拟屏幕点击（支持任意坐标精准触发）
  - 空格键：启动 / 结束拖动模式，配合方向键实现滑动操作（默认滑动时长 300ms，可在`swipe`函数中调整）
  - ESC 键：取消当前拖动操作

- **快捷按键集**：

| 按键 | 功能描述     | 对应 ADB 指令       |
| ---- | ------------ | ------------------- |
| p    | 电源键       | `input keyevent 26` |
| +    | 音量增加     | `input keyevent 24` |
| -    | 音量减少     | `input keyevent 25` |
| h    | Home 键      | `input keyevent 3`  |
| b    | 返回键       | `input keyevent 4`  |
| m    | 菜单键       | `input keyevent 82` |
| /    | 降低移动速度 | 调整步长（1-5 级）  |
| *    | 提高移动速度 | 调整步长（1-5 级）  |

### 3. 自定义按键系统

- **录制功能**：通过`C+数字键（1-9）`触发录制，5 秒内在设备上按下目标按键（支持物理按键如相机快门、自定义功能键），工具自动捕获键值并存储。
- **触发机制**：使用`c+数字键（1-9）`调用已录制的按键，对于特殊键（如相机对焦键 766、快门键 766）采用`sendevent`指令实现底层触发，兼容非标准 Android 按键。

### 4. 双模式切换

- 操作模式（Ctrl+O 进入）

  ：专注于屏幕交互，整合预览框、指针控制、按键模拟功能，适合需要精准定位的操作（如点击特定按钮、滑动界面）。支持两种移动模式：

  - 实时移动：指针随方向键持续移动（速度可调）
  - 步进移动：每按一次方向键移动固定步长

- **输入模式（Ctrl+I 进入）**：实时文本传输系统，支持中英文输入、退格修正（对应键值 67）、回车提交（对应键值 66），输入内容即时同步至设备输入框。

### 5. 显示优化选项

通过环境变量可灵活调整预览效果：

- `TERM_CHAR_ASPECT_RATIO`：自定义终端字符宽高比（默认 0.5），解决不同终端字体导致的比例失真
- `TERM_BOX_SCALE`：调整预览框缩放系数（默认 0.8），平衡显示面积与终端空间
- `TERM_DENSE_FILL=1`：启用紧凑模式，以 "・" 替代空格填充预览区域，提升屏幕细节辨识度
- `TERM_GRID_CHAR`：自定义网格线字符（默认 "・"）
- `TERM_GRID_INTERVAL`：设置网格线间隔密度（默认 5 字符）

## 适用场景

- 设备屏幕损坏但需临时操作
- 自动化测试中的精准坐标点击
- 无图形界面环境下的设备控制
- 特殊按键功能的快速映射与调用

## 版本选择

| **操作系统** | **架构**              | **文件名示例**                 |
| ------------ | --------------------- | ------------------------------ |
| **Windows**  | x86_64/AMD64          | `abc_vx.x.x_windows_amd64.exe` |
|              | ARM64                 | `abc_vx.x.x_windows_arm64.exe` |
| **Linux**    | x86_64/AMD64          | `abc_vx.x.x_linux_amd64`       |
|              | ARM64/AArch64         | `abc_vx.x.x_linux_arm64`       |
|              | Loong64               | `abc_vx.x.x_linux_loong64`     |
| **macOS**    | x86_64/AMD64          | `abc_vx.x.x_darwin_amd64`      |
|              | ARM64 (Apple Silicon) | `abc_vx.x.x_darwin_arm64`      |

> **问:** 如何确定我的操作系统架构？
> **答:** 执行相应命令查看：
>
> - Linux/macOS: `arch` 或 `uname -m`
> - Windows: 在 PowerShell 中执行 `[Environment]::Is64BitOperatingSystem` 查看是否为 64 位系统，通过设备管理器查看处理器架构
>
> > x86_64/amd64 → AMD64 架构
> > aarch64/arm64 → ARM64 架构
> > loong64 → 龙芯 64 位架构

## 许可证

ABC 采用 MIT 许可证授权：

```plaintext
MIT License

Copyright (c) 2025 Geekstrange

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```
