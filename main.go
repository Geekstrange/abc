package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/eiannone/keyboard"
	"golang.org/x/term"
)

// 屏幕尺寸和指针设置
var (
	screenWidth, screenHeight int = 1080, 1920
	currentX, currentY        int = 540, 960 // 初始位置 (屏幕中心)
	pointerChar               string = "●"
	moveStep                  int    = 15
	dragStartX, dragStartY    int
	isDragging                bool
	customKeys                map[string]string // 存储自定义键绑定 c1->keycode
)

// 初始化自定义键映射
func init() {
	customKeys = make(map[string]string)
}

// 执行ADB命令并返回结果
func runAdbCommand(args ...string) (string, error) {
	cmd := exec.Command("adb", args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// 获取屏幕尺寸
func getScreenSize() error {
	output, err := runAdbCommand("shell", "wm", "size")
	if err != nil {
		return err
	}

	// 解析输出: "Physical size: 1080x1920"
	re := regexp.MustCompile(`(\d+)x(\d+)`)
	matches := re.FindStringSubmatch(string(output))
	if len(matches) < 3 {
		return fmt.Errorf("无法解析屏幕尺寸: %s", output)
	}

	width, _ := strconv.Atoi(matches[1])
	height, _ := strconv.Atoi(matches[2])
	screenWidth = width
	screenHeight = height
	currentX, currentY = screenWidth/2, screenHeight/2
	return nil
}

// 获取终端尺寸
func getTerminalSize() (int, int, error) {
	fd := int(os.Stdout.Fd())
	width, height, err := term.GetSize(fd)
	if err != nil {
		return 0, 0, fmt.Errorf("终端尺寸获取失败: %w", err)
	}
	return width, height, nil
}

// 获取字符宽高比 (宽/高)
func getCharAspectRatio() float64 {
	if ratioStr := os.Getenv("TERM_CHAR_ASPECT_RATIO"); ratioStr != "" {
		if ratio, err := strconv.ParseFloat(ratioStr, 64); err == nil {
			return ratio
		}
	}
	return 0.5 // 默认值 (宽/高=1/2)
}

// 计算显示尺寸 (添加缩放因子控制大小)
func calculateBoxSize(devW, devH, termW, termH int, charRatio float64) (int, int) {
	// 添加缩放因子 (默认0.8 = 80%大小)
	scaleFactor := 0.8
	if scaleStr := os.Getenv("TERM_BOX_SCALE"); scaleStr != "" {
		if s, err := strconv.ParseFloat(scaleStr, 64); err == nil {
			scaleFactor = s
		}
	}

	// 应用缩放因子到终端尺寸
	termW = int(float64(termW) * scaleFactor)
	termH = int(float64(termH) * scaleFactor)

	// 保留边框空间 (左右各1字符+边框)
	maxWidth := termW - 4
	maxHeight := termH - 4
	if maxWidth < 8 {
		maxWidth = 8
	}
	if maxHeight < 8 {
		maxHeight = 8
	}

	// 设备宽高比
	devAspect := float64(devW) / float64(devH)

	// 计算字符宽高比修正后的高度
	calcHeight := int(float64(maxWidth) / devAspect * charRatio)
	if calcHeight <= maxHeight {
		return maxWidth, calcHeight
	}

	// 若高度超出则反向计算宽度
	calcWidth := int(float64(maxHeight) * devAspect / charRatio)
	if calcWidth > maxWidth {
		return maxWidth, maxHeight
	}
	return calcWidth, maxHeight
}

// 模拟点击
func tap(x, y int) error {
	_, err := runAdbCommand("shell", "input", "tap", strconv.Itoa(x), strconv.Itoa(y))
	return err
}

// 模拟滑动
func swipe(x1, y1, x2, y2, duration int) error {
	_, err := runAdbCommand("shell", "input", "swipe",
		strconv.Itoa(x1), strconv.Itoa(y1),
		strconv.Itoa(x2), strconv.Itoa(y2),
		strconv.Itoa(duration))
	return err
}

// 模拟按键 (增强版, 支持特殊按键)
func pressKey(keyCode string) error {
	// 特殊按键列表, 需要使用sendevent
	specialKeys := map[string]bool{
		"766": true, // 快门键
		"800": true, // 对焦键
	}

	if specialKeys[keyCode] {
		// 查找输入设备 (通常是/event1或/event2)
		devices, err := runAdbCommand("shell", "ls /dev/input/event*")
		if err != nil {
			return err
		}
		device := strings.Fields(devices)[0] // 取第一个输入设备

		// 发送按下和释放事件
		_, err = runAdbCommand("shell",
			fmt.Sprintf("sendevent %s 1 %s 1 && sleep 0.1 && sendevent %s 1 %s 0",
				device, keyCode, device, keyCode))
		return err
	}

	// 普通按键使用input keyevent
	_, err := runAdbCommand("shell", "input", "keyevent", keyCode)
	return err
}

// 发送文本
func sendText(text string) error {
	text = strings.ReplaceAll(text, " ", "%s")
	_, err := runAdbCommand("shell", "input", "text", text)
	return err
}

// 清屏
func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

// 显示操作模式界面
func displayOperationUI() {
	clearScreen()
	fmt.Println("===== 操作模式 =====")
	fmt.Println("Ctrl+I: 切换到输入模式 | Ctrl+C: 退出")
	fmt.Println("-----------------------------------")
	fmt.Println("方向键: 移动指针 | Enter: 点击")
	fmt.Println("空格: 开始/结束拖动 | ESC: 取消拖动")
	fmt.Println("p:电源 | +:音量加 | -:音量减 | h:Home | b:返回 | m:菜单")
	fmt.Println("c1-c9: 自定义键(按对应键触发) | C1-C9: 录制自定义键")
	fmt.Println("-----------------------------------")
	displayPointerArea()
	fmt.Printf("\n当前位置: (%d, %d)\n", currentX, currentY)
	if len(customKeys) > 0 {
		fmt.Print("已绑定自定义键: ")
		for k := range customKeys {
			fmt.Printf("%s ", k)
		}
		fmt.Println()
	}
	fmt.Print("请输入操作: ")
}

// 显示按手机实际比例的指针区域
func displayPointerArea() {
	// 获取设备屏幕尺寸
	phoneW := screenWidth
	phoneH := screenHeight

	// 获取终端尺寸
	termW, termH, err := getTerminalSize()
	if err != nil {
		fmt.Printf("获取终端尺寸失败: %v\n", err)
		termW, termH = 80, 24 // 使用默认值
	}

	// 获取字符宽高比 (宽/高)
	charAspect := getCharAspectRatio()

	// 计算预览框尺寸
	boxWidth, boxHeight := calculateBoxSize(phoneW, phoneH, termW, termH, charAspect)

	// 指针在网格中的坐标
	gx := (currentX * boxWidth) / phoneW
	gy := (currentY * boxHeight) / phoneH

	// 填充字符 (支持紧凑模式)
	fillChar := " "
	if os.Getenv("TERM_DENSE_FILL") == "1" {
		fillChar = "·" // Unicode U+00B7 减少纵向拉伸感
	}

	fmt.Printf("屏幕预览 (手机 %d:%d, 字符宽高比 %.1f)\n",
		phoneW, phoneH, charAspect)
	fmt.Print("+")
	fmt.Print(strings.Repeat("-", boxWidth))
	fmt.Println("+")

	for y := 0; y < boxHeight; y++ {
		fmt.Print("|")
		for x := 0; x < boxWidth; x++ {
			if x == gx && y == gy {
				fmt.Print(pointerChar)
			} else {
				fmt.Print(fillChar)
			}
		}
		fmt.Println("|")
	}

	fmt.Print("+")
	fmt.Print(strings.Repeat("-", boxWidth))
	fmt.Println("+")
}

// 修改监听按键事件的函数, 正确解析键值
func listenForKeyPress() (string, string, error) {
	fmt.Println("请在手机上按下并释放要绑定的按键 (5秒内)...")

	cmd := exec.Command("adb", "shell", "getevent", "-l")
	output, err := cmd.StdoutPipe()
	if err != nil {
		return "", "", err
	}
	if err := cmd.Start(); err != nil {
		return "", "", err
	}

	done := make(chan struct {
		keyHex  string
		keyCode string
	}, 1)
	errChan := make(chan error, 1)

	go func() {
		buf := make([]byte, 1024)
		var lastDownKey string

		for {
			n, err := output.Read(buf)
			if err != nil {
				errChan <- err
				return
			}

			line := string(buf[:n])
			if strings.Contains(line, "DOWN") {
				parts := strings.Fields(line)
				if len(parts) >= 3 {
					lastDownKey = parts[2]
				}
			} else if strings.Contains(line, "UP") && lastDownKey != "" {
				// 按键名称到键码的映射表
				keyMap := map[string]string{
					"KEY_VOLUMEUP":    "24",
					"KEY_VOLUMEDOWN":  "25",
					"KEY_POWER":       "26",
					"KEY_CAMERA":      "27",    // 基础相机键
					"KEY_FOCUS":       "800",   // 对焦键
					"KEY_CAMERA_SNAP": "766",   // 快门键
				}

				// 先检查是否是已知按键名称
				if code, ok := keyMap[lastDownKey]; ok {
					done <- struct {
						keyHex  string
						keyCode string
					}{lastDownKey, code}
					return
				}

				// 尝试解析为十六进制数字
				keyCodeInt, err := strconv.ParseInt(lastDownKey, 16, 32)
				if err == nil {
					keyCode := strconv.Itoa(int(keyCodeInt))
					done <- struct {
						keyHex  string
						keyCode string
					}{lastDownKey, keyCode}
					return
				}

				// 无法识别的按键
				done <- struct {
					keyHex  string
					keyCode string
				}{lastDownKey, lastDownKey}
				return
			}
		}
	}()

	select {
	case result := <-done:
		cmd.Process.Kill()
		return result.keyHex, result.keyCode, nil
	case err := <-errChan:
		cmd.Process.Kill()
		return "", "", err
	case <-time.After(5 * time.Second):
		cmd.Process.Kill()
		return "", "", fmt.Errorf("超时未检测到按键")
	}
}

// 操作模式
func operationMode() error {
	displayOperationUI()
	for {
		char, key, err := keyboard.GetKey()
		if err != nil {
			return err
		}

		// Ctrl+I 切换输入模式
		if key == keyboard.KeyCtrlI {
			fmt.Println("\n切换到输入模式...")
			return nil
		}

		switch key {
		case keyboard.KeyArrowUp:
			currentY = max(0, currentY-moveStep)
			displayOperationUI()
		case keyboard.KeyArrowDown:
			currentY = min(screenHeight, currentY+moveStep)
			displayOperationUI()
		case keyboard.KeyArrowLeft:
			currentX = max(0, currentX-moveStep)
			displayOperationUI()
		case keyboard.KeyArrowRight:
			currentX = min(screenWidth, currentX+moveStep)
			displayOperationUI()
		case keyboard.KeyEnter:
			if !isDragging {
				fmt.Println("\n执行点击...")
				if err := tap(currentX, currentY); err != nil {
					fmt.Println("点击失败:", err)
				} else {
					fmt.Println("点击成功")
				}
				time.Sleep(300 * time.Millisecond)
				displayOperationUI()
			}
		case keyboard.KeySpace:
			if !isDragging {
				dragStartX, dragStartY = currentX, currentY
				isDragging = true
				fmt.Println("\n开始拖动 (移动指针后按空格结束拖动, ESC取消)")
				displayOperationUI()
			} else {
				fmt.Println("\n结束拖动...")
				if err := swipe(dragStartX, dragStartY, currentX, currentY, 300); err != nil {
					fmt.Println("拖动失败:", err)
				} else {
					fmt.Println("拖动成功")
				}
				isDragging = false
				time.Sleep(300 * time.Millisecond)
				displayOperationUI()
			}
		case keyboard.KeyEsc:
			if isDragging {
				fmt.Println("\n已取消拖动")
				isDragging = false
				time.Sleep(300 * time.Millisecond)
				displayOperationUI()
			}
		case keyboard.KeyCtrlC:
			fmt.Println("\n退出程序...")
			os.Exit(0)
		}

		// 处理自定义键绑定 (使用大写C表示录制)
		if char == 'C' {
			// 等待输入数字1-9
			fmt.Print("\n请输入要绑定的编号(1-9): ")
			// 关闭回显, 避免输入数字时显示额外字符
			exec.Command("stty", "-echo").Run()
			defer exec.Command("stty", "echo").Run() // 确保最终恢复回显

			// 循环等待有效的数字输入
			var subChar rune
			for {
				c, k, err := keyboard.GetKey()
				if err != nil {
					fmt.Println("获取按键失败:", err)
					displayOperationUI()
					break
				}
				if k == keyboard.KeyCtrlC {
					fmt.Println("\n退出程序...")
					os.Exit(0)
				}
				if c >= '1' && c <= '9' {
					subChar = c
					break
				}
				fmt.Print("\r请输入有效的编号(1-9): ")
			}

			fmt.Println(subChar) // 显示用户输入的数字
			keyHex, keyCode, err := listenForKeyPress()
			if err != nil {
				fmt.Println("捕获按键失败:", err)
			} else {
				key := "c" + string(subChar)
				customKeys[key] = keyCode
				fmt.Printf("捕获键值: 0x%s -> 十进制: %s\n", keyHex, keyCode)
				fmt.Printf("已将按键绑定到 %s\n", key)
			}
			time.Sleep(1000 * time.Millisecond)
			displayOperationUI()
		} else {
			switch char {
			case 'p':
				fmt.Println("\n模拟电源键...")
				pressKey("26")
				time.Sleep(300 * time.Millisecond)
				displayOperationUI()
			case '+':
				fmt.Println("\n模拟音量加...")
				pressKey("24")
				time.Sleep(300 * time.Millisecond)
				displayOperationUI()
			case '-':
				fmt.Println("\n模拟音量减...")
				pressKey("25")
				time.Sleep(300 * time.Millisecond)
				displayOperationUI()
			case 'h':
				fmt.Println("\n模拟Home键...")
				pressKey("3")
				time.Sleep(300 * time.Millisecond)
				displayOperationUI()
			case 'b':
				fmt.Println("\n模拟返回键...")
				pressKey("4")
				time.Sleep(300 * time.Millisecond)
				displayOperationUI()
			case 'm':
				fmt.Println("\n模拟菜单键...")
				pressKey("82")
				time.Sleep(300 * time.Millisecond)
				displayOperationUI()
			// 处理自定义键触发 (小写字母表示触发)
			case 'c':
				// 等待输入数字1-9
				fmt.Print("\n请输入要触发的编号(1-9): ")
				subChar, _, err := keyboard.GetKey()
				if err != nil || subChar < '1' || subChar > '9' {
					fmt.Println("无效输入")
					time.Sleep(1000 * time.Millisecond)
					displayOperationUI()
					break
				}
				key := "c" + string(subChar)
				if code, exists := customKeys[key]; exists {
					fmt.Printf("触发自定义键 %s (code: %s)\n", key, code)
					pressKey(code)
				} else {
					fmt.Printf("未绑定的自定义键 %s\n", key)
				}
				time.Sleep(1000 * time.Millisecond)
				displayOperationUI()
			}
		}
	}
}

// 输入模式帮助信息
func printInputHelp() {
	fmt.Println("\n===== 输入模式 =====")
	fmt.Println("输入的内容将实时发送到手机")
	fmt.Println("退格键: 删除最后一个字符")
	fmt.Println("Enter: 发送回车")
	fmt.Println("Ctrl+O: 切换到操作模式")
	fmt.Println("Ctrl+C: 退出程序")
	fmt.Println("====================")
}

// 输入模式
func inputMode() error {
	fmt.Println("已进入输入模式 (Ctrl+O切换到操作模式)")
	printInputHelp()
	fmt.Print("输入内容: ")

	var inputBuffer strings.Builder
	for {
		char, key, err := keyboard.GetKey()
		if err != nil {
			return err
		}

		if key == keyboard.KeyCtrlO {
			fmt.Println("\n切换到操作模式...")
			return nil
		}

		switch key {
		case keyboard.KeyCtrlC:
			fmt.Println("\n退出程序...")
			os.Exit(0)
		case keyboard.KeyBackspace:
			if inputBuffer.Len() > 0 {
				str := inputBuffer.String()
				inputBuffer.Reset()
				inputBuffer.WriteString(str[:len(str)-1])
				fmt.Print("\r输入内容: " + inputBuffer.String() + " ")
				fmt.Print("\r输入内容: " + inputBuffer.String())
				pressKey("67")
			}
		case keyboard.KeyEnter:
			inputBuffer.WriteRune('\n')
			fmt.Print("\r输入内容: " + inputBuffer.String())
			pressKey("66")
		default:
			if char != 0 {
				inputBuffer.WriteRune(char)
				fmt.Print("\r输入内容: " + inputBuffer.String())
				sendText(string(char))
			}
		}
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func main() {
	if _, err := exec.LookPath("adb"); err != nil {
		fmt.Println("错误: 未找到ADB请确保ADB已安装并添加到环境变量中")
		os.Exit(1)
	}

	output, err := runAdbCommand("devices")
	if err != nil {
		fmt.Println("ADB命令执行失败:", err)
		os.Exit(1)
	}
	if !strings.Contains(output, "device") {
		fmt.Println("错误: 未检测到连接的设备请确保手机已通过USB连接并启用了调试模式")
		os.Exit(1)
	}

	if err := getScreenSize(); err != nil {
		fmt.Println("警告: 获取屏幕尺寸失败, 使用默认值(1080x1920)错误:", err)
	}

	if err := keyboard.Open(); err != nil {
		panic(err)
	}
	defer keyboard.Close()

	fmt.Println("ADB Basic Contorller")
	fmt.Println("Ctrl+O: 进入操作模式")
	fmt.Println("Ctrl+I: 进入输入模式")
	fmt.Println("Ctrl+C: 退出程序")

	for {
		_, key, err := keyboard.GetKey()
		if err != nil {
			panic(err)
		}

		switch key {
		case keyboard.KeyCtrlO:
			if err := operationMode(); err != nil {
				fmt.Println("操作模式出错:", err)
			}
		case keyboard.KeyCtrlI:
			if err := inputMode(); err != nil {
				fmt.Println("输入模式出错:", err)
			}
		case keyboard.KeyCtrlC:
			fmt.Println("\n退出程序...")
			return
		default:
			fmt.Println("\n请按 Ctrl+O 进入操作模式或 Ctrl+I 进入输入模式")
		}
	}
}
