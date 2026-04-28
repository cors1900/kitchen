package fmt

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"atomicgo.dev/cursor"
	"github.com/mattn/go-runewidth"
	"golang.org/x/term"
)

type ProgressBar struct {
	lock      sync.Mutex
	startTime time.Time
	taskName  string

	unit        string
	showPercent bool
	showSize    bool
	showSpeed   bool
	showETA     bool
	autoWidth   bool // 是否开启自适应
	fixedWidth  int  // 手动指定的宽度
	finishMsg   string

	current int64
	total   int64

	stop    chan bool
	stopped bool
	wg      sync.WaitGroup

	symbols     []rune
	symbolIndex int
}

// --- Options ---

func WithSize(unit string) Option { return func(p *ProgressBar) { p.showSize = true; p.unit = unit } }
func WithSpeed() Option           { return func(p *ProgressBar) { p.showSpeed = true } }
func WithETA() Option             { return func(p *ProgressBar) { p.showETA = true } }
func WithWidth(w int) Option      { return func(p *ProgressBar) { p.fixedWidth = w; p.autoWidth = false } }
func WithFinishMsg(msg string, args ...any) Option {
	return func(p *ProgressBar) { p.finishMsg = Sprintf(msg, args...) }
}

type Option func(*ProgressBar)

func NewProgressBar(name string, opts ...Option) *ProgressBar {
	p := &ProgressBar{
		taskName:    name,
		startTime:   time.Now(),
		autoWidth:   true, // 默认自适应
		showPercent: true,
		// finishMsg:   "✔ Done",
		unit:    "MB",
		symbols: []rune{'⢿', '⣻', '⣽', '⣾', '⣷', '⣯', '⣟', '⡿'},

		stop: make(chan bool, 1),
	}
	for _, opt := range opts {
		opt(p)
	}
	p.start()
	return p
}

func (p *ProgressBar) start() {
	p.wg.Add(1)
	go func() {
		cursor.Hide()
		defer cursor.Show()
		defer p.wg.Done()
		ticker := time.NewTicker(120 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-p.stop:
				return
			case <-ticker.C:
				{
					p.lock.Lock()
					p.symbolIndex = (p.symbolIndex + 1) % len(p.symbols)
					p.render()
					p.lock.Unlock()
					if p.current >= p.total {
						break
					}
				}
			}
		}
	}()
}

func (p *ProgressBar) Stop() {
	if p.stopped {
		return
	}

	select {
	case p.stop <- true:
		// 发送成功
	default:
		// 通道已满，忽略
	}

	p.stopped = true
	p.wg.Wait()
}

func (p *ProgressBar) SetName(name string) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.taskName = name
}

func (p *ProgressBar) Name() string {
	p.lock.Lock()
	defer p.lock.Unlock()
	return p.taskName
}

func (p *ProgressBar) Update(current int64, total int64) bool {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.current = current
	if total <= 0 {
		total = 1
	}
	p.total = total
	p.render()
	if p.current >= p.total {
		p.Stop()

		if p.finishMsg != "" {
			Info("\r%s", p.finishMsg)
		} else {
			Printf("\n")
		}

		return true
	}
	return false
}

func (p *ProgressBar) getTerminalWidth() int {
	if !p.autoWidth {
		return p.fixedWidth
	}

	// 获取终端宽度
	fd := int(os.Stdout.Fd())
	tw, _, err := term.GetSize(fd)
	if err != nil {
		return 25 // 回退默认值
	}
	return tw
}

func (p *ProgressBar) sizeString() string {
	MB := func(s int64) float64 {
		return float64(s) / 1024 / 1024
	}
	KB := func(s int64) float64 {
		return float64(s) / 1024
	}

	switch p.unit {
	case "MB":
		return fmt.Sprintf("%.1f/%.1f MB", MB(p.current), MB(p.total))
	case "KB":
		return fmt.Sprintf("%.1f/%.1f KB", KB(p.current), KB(p.total))
	default:
		return fmt.Sprintf("%v/%v", p.current, p.total)
	}
}

func (p *ProgressBar) render() {
	ratio := float64(p.current) / float64(p.total)

	// 1. 先构建右侧信息字符串，用于计算剩余空间
	var info []string

	if p.showPercent {
		info = append(info, fmt.Sprintf("%.0f%%", ratio*100))
	}
	if p.showSize {
		info = append(info, p.sizeString())
	}

	elapsed := time.Since(p.startTime).Seconds()
	if elapsed > 0 {
		speed := float64(p.current) / elapsed
		if p.showSpeed {
			info = append(info, fmt.Sprintf("%s/s", Size(int64(speed))))
		}
		if p.showETA && speed > 0 && p.current < p.total {
			remaining := float64(p.total-p.current) / speed
			info = append(info, fmt.Sprintf("%s", Duration(time.Duration(remaining)*time.Second)))
		}
	}

	infoStr := strings.Join(info, " | ")
	if p.current < p.total {
		// infoStr += " "
		// infoStr += string(p.symbols[p.symbolIndex])
	} else {
		// infoStr += " "
		// infoStr += "⣿"
	}

	terminalWidth := p.getTerminalWidth()
	taskNameWidth := runewidth.StringWidth(p.taskName)
	symbolWidth := runewidth.RuneWidth(p.symbols[p.symbolIndex])
	infoStrLen := runewidth.StringWidth(infoStr)

	padding := strings.Repeat(" ", max(0, terminalWidth-runewidth.StringWidth(appName)-symbolWidth-taskNameWidth-infoStrLen-2))

	// 打印进度条
	Infof("\r%c %s %s%s", p.symbols[p.symbolIndex], p.taskName, padding, infoStr)

	// [" ", "⡀", "⡄", "⡆", "⡇", "⡏", "⡟", "⡿", "⣿"]
}
