# GUI 界面使用指南

本文档说明如何在 Windows 和 Linux 环境下运行 gui 程序，包括 CLI 模式（默认）和桌面 GUI 模式（Fyne）。

---

## 两种运行模式

gui 程序支持两种模式，通过 **Go build tag** 在编译时选择，同一份代码生成两种不同的二进制文件：

| 模式 | Build Tag | 界面 | 依赖 | 适用场景 |
|------|-----------|------|------|----------|
| **CLI 模式**（默认） | 无（`!fyne`） | 终端交互 | 无额外依赖 | SSH 远程、服务器环境、自动化 |
| **桌面 GUI 模式** | `-tags fyne` | 480×560 图形窗口 | CGO + Fyne 库 | Windows/Linux 桌面环境 |

两者**互斥**，不能合并为同一个二进制文件。

---

## CLI 模式（当前默认）

### 编译

```bash
cd gui
go build -o dist/gui ./cmd/gui/
```

或使用 Makefile：

```bash
make build
```

### 运行

```bash
./gui/dist/gui --config configs/gui.yaml
```

### 交互方式

程序在终端内全程文字交互：

```
remote-gui (CLI mode — build with -tags fyne for desktop UI)
Available operations:
  [1] 查询消息 → query-rocketmq-msg

Enter operation number (or 'q' to quit): 1
  topic [输入 Topic]: MY_TOPIC
  msgId [输入 MsgId]: abc123
Executing '查询消息'...
Status: success  ExitCode: 0  Duration: 342ms
--- stdout ---
<查询结果输出>
```

可附加 `-debug` 查看详细日志：

```bash
./gui/dist/gui --config configs/gui.yaml --debug
```

---

## 桌面 GUI 模式（Fyne 窗口）

Fyne 使用 CGO，编译时需要 C 编译器和系统图形库。

### Windows

#### 1. 安装前提条件

**Go 1.21+**（已安装则跳过）：[golang.org/dl](https://golang.org/dl/)

**MinGW-w64**（提供 CGO 所需的 gcc）：

推荐使用 [MSYS2](https://www.msys2.org/)：

```powershell
# 在 MSYS2 MINGW64 终端中执行
pacman -S mingw-w64-x86_64-gcc
```

安装完成后，将 `C:\msys64\mingw64\bin` 添加到系统 PATH。

验证 gcc 可用：

```cmd
gcc --version
```

#### 2. 添加 Fyne 依赖

```cmd
cd gui
go get fyne.io/fyne/v2@latest
```

#### 3. 编译桌面 GUI 版本

```cmd
cd gui
go build -tags fyne -o dist\remote-gui.exe .\cmd\gui\
```

#### 4. 运行

```cmd
dist\remote-gui.exe --config configs\gui.yaml
```

双击 `remote-gui.exe` 也可直接打开（需确保当前目录或环境变量中能找到 `configs\gui.yaml`）。

窗口大小 480×560，包含操作下拉框、参数输入框、执行按钮和结果显示区。

---

### Linux（有桌面环境 / X11）

适用于安装了 GNOME、KDE、XFCE 等桌面环境的 Linux 系统。

#### 1. 安装系统依赖

**Debian / Ubuntu：**

```bash
sudo apt update
sudo apt install -y gcc libgl1-mesa-dev xorg-dev
```

**RHEL / CentOS / Fedora：**

```bash
sudo dnf install -y gcc mesa-libGL-devel libX11-devel libXrandr-devel \
    libXinerama-devel libXcursor-devel libXi-devel
```

**Arch Linux：**

```bash
sudo pacman -S gcc mesa libx11 libxrandr libxinerama libxcursor libxi
```

#### 2. 添加 Fyne 依赖

```bash
cd gui
go get fyne.io/fyne/v2@latest
```

#### 3. 编译桌面 GUI 版本

```bash
cd gui
go build -tags fyne -o dist/remote-gui ./cmd/gui/
```

#### 4. 运行

```bash
./gui/dist/remote-gui --config configs/gui.yaml
```

---

### Linux（无桌面环境 / SSH 服务器）

在纯命令行服务器上运行 Fyne 窗口有两种方案：

#### 方案一：SSH X11 转发（推荐，窗口显示在本地）

**前提**：本地机器安装 X Server：
- macOS：安装 [XQuartz](https://www.xquartz.org/)
- Windows：安装 [VcXsrv](https://sourceforge.net/projects/vcxsrv/) 或 [MobaXterm](https://mobaxterm.mobatek.net/)
- Linux：已内置 X Server

**连接远程服务器时开启 X11 转发：**

```bash
ssh -X user@your-server-ip
```

若需更好的性能（压缩传输）：

```bash
ssh -Y user@your-server-ip   # 受信任模式，速度更快
```

**在远程服务器上运行 GUI（窗口弹出在本地）：**

```bash
./gui/dist/remote-gui --config configs/gui.yaml
```

验证 DISPLAY 已设置：

```bash
echo $DISPLAY   # 应输出类似 localhost:10.0 或 :0
```

#### 方案二：Xvfb 虚拟帧缓冲 + VNC（服务器端渲染，通过 VNC 查看）

适合需要在服务器端持久化显示或多人共享的场景。

```bash
# 安装依赖
sudo apt install -y xvfb x11vnc

# 启动虚拟帧缓冲（在后台）
Xvfb :99 -screen 0 1280x800x24 &

# 启动 VNC 服务器（可选，用于远程查看）
x11vnc -display :99 -nopw -listen 0.0.0.0 -port 5900 &

# 以虚拟 DISPLAY 运行 GUI
DISPLAY=:99 ./gui/dist/remote-gui --config configs/gui.yaml
```

通过 VNC 客户端（如 TigerVNC、RealVNC）连接 `your-server-ip:5900` 即可看到窗口。

---

## 故障排查

### `cgo: C compiler "gcc" not found`

**原因**：Fyne 需要 CGO，系统未安装 C 编译器。

**解决**：
- Linux：`sudo apt install gcc`
- Windows：安装 MinGW-w64 并将其 bin 目录加入 PATH

验证：`gcc --version`

---

### `Error: cannot connect to display` 或 `DISPLAY not set`

**原因**：Linux 无图形环境，Fyne 找不到 X11 Display。

**解决**：
- 若有桌面登录会话：确认 `echo $DISPLAY` 有输出（如 `:0`）
- 若通过 SSH 连接：使用 `ssh -X` 开启 X11 转发
- 若无桌面：使用 Xvfb 方案，并在运行前 `export DISPLAY=:99`

---

### `undefined reference to ...` 链接错误

**原因**：缺少图形库头文件或链接库。

**解决**（Debian/Ubuntu）：

```bash
sudo apt install -y libgl1-mesa-dev xorg-dev
```

---

### Windows 编译时 `cc1.exe: sorry, unimplemented`

**原因**：MinGW 版本过旧或与 Go 版本不匹配。

**解决**：使用 MSYS2 重新安装最新版 MinGW-w64：

```bash
pacman -Syu
pacman -S mingw-w64-x86_64-gcc
```

---

### `go: module fyne.io/fyne/v2 not found`

**原因**：Fyne 依赖尚未添加到 go.mod。

**解决**：

```bash
cd gui
go get fyne.io/fyne/v2@latest
go mod tidy
```

---

## 生产建议

| 场景 | 推荐模式 |
|------|----------|
| 开发者本地 Windows/Linux 桌面 | 桌面 GUI（`-tags fyne`） |
| 运维通过 SSH 远程执行 | CLI 模式（默认编译） |
| CI/CD 自动化调用 | CLI 模式（默认编译） |
| 需要图形界面但在服务器 | SSH X11 转发 + 桌面 GUI |
| 多人共享 GUI 会话 | Xvfb + VNC + 桌面 GUI |

CLI 模式无需额外依赖，可在任意 Linux/macOS/Windows 终端下直接使用，是服务器和自动化场景的首选。
