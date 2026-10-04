# lazygit（commit graph 改进版）

这是从 [jesseduffield/lazygit](https://github.com/jesseduffield/lazygit) fork 过来的版本，在保留 lazygit 全部功能的基础上，重新设计了 commits 面板里 commit graph 的显示方式。分支多、彼此之间又互相 merge 时，原来的图很难看懂，这个版本让每条分支的走向一目了然。

## 在 macOS 上安装

目前没有预编译的二进制，需要从源码 build，大约一分钟就能完成。

**1. 安装 Go**（需要 1.25 或更新的版本）

```bash
brew install go
```

**2. 下载源码**

```bash
git clone https://github.com/dysonlee/lazygit.git ~/lazygit-src
```

**3. build 并安装到 `~/.local/bin`**

```bash
mkdir -p ~/.local/bin && go build -C ~/lazygit-src -o ~/.local/bin/lazygit .
```

**4. 确认 `~/.local/bin` 在 PATH 里**

fish：

```bash
fish_add_path ~/.local/bin
```

zsh / bash：在 `~/.zshrc` 或 `~/.bashrc` 里加上

```bash
export PATH="$HOME/.local/bin:$PATH"
```

**5. 检查安装结果**

```bash
lazygit --version
```

如果之前用 Homebrew 装过官方的 lazygit，要么先 `brew uninstall lazygit`，要么确保 `~/.local/bin` 在 PATH 里排在 Homebrew 前面，否则运行的仍然是官方版本。

### 更新

```bash
git -C ~/lazygit-src pull && go build -C ~/lazygit-src -o ~/.local/bin/lazygit .
```

### 卸载

```bash
rm ~/.local/bin/lazygit
```

## 开启新的 commit graph

新的图默认关闭，不开启时和官方 lazygit 完全一样。在 macOS 上，lazygit 的配置文件位于 `~/Library/Application Support/lazygit/config.yml`，加上：

```yaml
git:
  log:
    graphStyle: lanes        # 新的图；classic 是原来的样子
    showWholeGraph: true     # 显示所有分支，而不只是当前分支
    order: date-order        # 按提交时间交错排列各分支的 commit
gui:
  nerdFontsVersion: "3"      # 装了 Nerd Font 时，节点和标签图标更清晰
```

lazygit 会自动重新加载配置，不需要重启。按 `+` 把 commits 面板放大，看到的图最完整。

## 改进了什么

- **每条分支固定在一列**：一条分支从上到下始终在同一列，不会左右跳动；commit 沿着自己的 first-parent 链往下延伸，merge 线弯进来、弯出去。
- **主分支是一条直线**：`master` / `main`（由 `git.mainBranches` 决定）的历史始终留在自己那一列，其他分支并入它。
- **每条分支一种颜色**：每条新分支轮换分配一个颜色，同一列里先后出现的分支颜色也不同。
- **更大的节点、更宽的间距**：普通 commit 是大圆点，merge commit 是线上的小圆点，有实际改动的 commit 更显眼。
- **merge 线连进已有的线**：如果已经有一条线通向同一个 parent，merge 线会直接连进去，图更窄。
- **branch / tag 标签列**：图的左边单独一列显示分支和 tag，颜色与分支一致。当前分支带 ✓；配合 Nerd Font 时，会用图标区分本地分支、远程分支和 tag；同一个 commit 上有多个 ref 时显示 `+N`。
- **长分支名智能缩短**：远程分支省略 `origin/`，`hotfix/ROAR-13955` 缩成 `h/ROAR-13955`，仍然放不下时才省略中间部分。
- **选中 commit 时显示它所在的分支**：本身没有 ref 的 commit 被选中时，标签列会显示它所在分支的标签，并用点线连到节点。
- **非当前分支的 commit 变暗**：从 HEAD 走不到的 commit 显示为暗色，当前分支的历史更突出。
- **commit message 对齐成一列**：按屏幕上可见的最宽的图对齐，方便浏览。
- **选中高亮不会切断其他线**：选中 commit 时只给它自己的线换成高亮色，其他分支的线保持完整。
- **进入 commit 后显示作者、日期和分支**：按 enter 查看 commit 的文件时，标题里会显示作者、日期和所在分支。
- **圆角交汇（可选）**：在 kitty 或 Ghostty 里，线条交汇处可以画成圆角。

## 配置项

除了上面"开启新的 commit graph"里的设置，还有这些可选项：

```yaml
git:
  log:
    # 线条交汇处画成圆角。只在 kitty、Ghostty 这类会自己绘制分支字符的终端里开启，
    # 其他终端里大多数字体没有这些字符
    useBranchDrawingGlyphs: true
    # 把从 HEAD 走不到的 commit 显示为暗色（默认 true）
    dimUnreachableCommits: true
  # 哪些分支算主分支，它们的历史会保持为一条直线
  mainBranches: [master, main]
gui:
  # 放大 commits 面板时不显示日期列，进入 commit 后标题里仍会显示日期
  showCommitDateInExpandedView: false
  # 设为 0 时不显示作者列，进入 commit 后标题里仍会显示作者
  commitAuthorShortLength: 0
  commitAuthorLongLength: 0
```

## 与 lazygit 的关系

这个 fork 只改动了 commit graph 及相关的显示，lazygit 的其他功能、快捷键和配置都和上游一致，使用方法请参考上游的文档：

- [lazygit README](https://github.com/jesseduffield/lazygit#readme)
- [配置说明](docs-master/Config.md)
- [快捷键](docs-master/keybindings)

lazygit 由 [Jesse Duffield](https://github.com/jesseduffield) 和社区贡献者开发，以 [MIT 许可证](LICENSE) 发布，这个 fork 沿用同一许可证。
