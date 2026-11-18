// GoTrace - công cụ dòng lệnh truy vết thời gian thực thao tác người dùng và
// phản ứng của Windows Kernel, cho phép mở rộng từng dòng log để xem chi tiết
// Win32 API, subsystem và trạng thái CPU/Bộ nhớ thật.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"gotrace/pkg/control"
	"gotrace/pkg/monitor"
	"gotrace/pkg/store"
	"gotrace/pkg/ui"
	"gotrace/pkg/win32"
)

const (
	userPollInterval   = 150 * time.Millisecond
	kernelPollInterval = 250 * time.Millisecond
	maxLogEntries      = 1000

	// defaultIgnoreList là các tiến trình sống rất ngắn do công cụ khác spawn liên tục
	// (tasklist/conhost), che mất sự kiện cần đọc. Đổi bằng cờ -ignore.
	defaultIgnoreList = "tasklist.exe,conhost.exe"
)

func main() {
	ignore := flag.String("ignore", defaultIgnoreList,
		"tên tiến trình bị lọc khỏi log, cách nhau bởi dấu phẩy (để trống = không lọc)")
	flag.Parse()

	console, err := win32.NewConsole()
	if err != nil {
		fmt.Fprintln(os.Stderr, "GoTrace: không mở được console:", err)
		os.Exit(1)
	}
	defer console.Restore()

	logs := store.New(maxLogEntries)
	ctrl := control.New(strings.Split(*ignore, ","))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go (&monitor.UserActionTracker{Store: logs, Interval: userPollInterval, Control: ctrl}).Run(ctx)
	go (&monitor.OSReactionWatcher{Store: logs, Interval: kernelPollInterval, Control: ctrl}).Run(ctx)

	// Ctrl+C: huỷ context để cả UI và hai monitor thoát sạch.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	go func() {
		<-interrupt
		cancel()
	}()

	app := ui.New(logs, console, ctrl)
	app.PollUser = userPollInterval
	app.PollKernel = kernelPollInterval
	if err := app.Run(ctx); err != nil {
		console.Restore()
		fmt.Fprintln(os.Stderr, "\nGoTrace gặp lỗi:", err)
		os.Exit(1)
	}
}
