// Package main 调试互斥体功能
// 直接测试Windows互斥体API，不依赖其他逻辑
package main

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"
)

const (
	ERROR_ALREADY_EXISTS = 183
)

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	procCreateMutex  = kernel32.NewProc("CreateMutexW")
	procGetLastError = kernel32.NewProc("GetLastError")
)

func main() {
	fmt.Printf("🧪 调试互斥体功能 - PID: %d, 时间: %s\n", syscall.Getpid(), time.Now().Format("15:04:05"))

	mutexName := "Global\\TestMutex_Debug"
	fmt.Printf("🏷️ 互斥体名称: %s\n", mutexName)

	// 转换为Windows宽字符串
	mutexNamePtr, err := syscall.UTF16PtrFromString(mutexName)
	if err != nil {
		fmt.Printf("❌ 转换字符串失败: %v\n", err)
		return
	}

	fmt.Println("🔍 正在调用 CreateMutex...")
	// 创建互斥体
	ret, _, callErr := procCreateMutex.Call(
		0,                                     // 安全属性
		0,                                     // 初始所有者
		uintptr(unsafe.Pointer(mutexNamePtr)), // 互斥体名称
	)

	fmt.Printf("📈 CreateMutex 返回: ret=%d, callErr=%v\n", ret, callErr)

	if ret == 0 {
		fmt.Println("❌ CreateMutex 调用失败")
		return
	}

	// 获取错误码
	lastError, _, _ := procGetLastError.Call()

	handle := syscall.Handle(ret)
	isFirst := lastError != ERROR_ALREADY_EXISTS

	fmt.Printf("📊 互斥体信息:\n")
	fmt.Printf("   句柄: %d\n", handle)
	fmt.Printf("   错误码: %d\n", lastError)
	fmt.Printf("   ERROR_ALREADY_EXISTS: %d\n", ERROR_ALREADY_EXISTS)
	fmt.Printf("   是否首个实例: %t\n", isFirst)

	if isFirst {
		fmt.Println("✅ 成功获取互斥体，作为首个实例")
		fmt.Println("💡 现在尝试运行第二个实例来测试")
		fmt.Println("🔄 程序将运行30秒...")

		// 保持程序运行，定期输出状态
		for i := 0; i < 30; i++ {
			fmt.Printf("📍 程序运行中... %d/%d\n", i+1, 30)
			time.Sleep(1 * time.Second)
		}
	} else {
		fmt.Println("⚠️ 互斥体已存在，这是后续实例")
		fmt.Println("💬 这表示单实例机制正常工作！")
	}

	fmt.Println("👋 程序退出")
}
