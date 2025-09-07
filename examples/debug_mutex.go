// Package main 互斥体调试程序
// 专门用于调试Windows互斥体功能
package main

import (
	"fmt"
	"log"
	"os"
	"syscall"
	"time"
	"unsafe"
)

// Windows API 常量定义
const (
	ERROR_ALREADY_EXISTS = 183 // 对象已存在错误码常量
)

// Windows API 函数声明
var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll") // 加载内核动态链接库
	procCreateMutex  = kernel32.NewProc("CreateMutexW")   // 创建互斥体API函数
	procCloseHandle  = kernel32.NewProc("CloseHandle")    // 关闭句柄API函数
	procGetLastError = kernel32.NewProc("GetLastError")   // 获取最后错误API函数
)

func main() {
	fmt.Printf("🧪 互斥体调试程序启动 (PID: %d)\n", os.Getpid())

	// 互斥体名称
	mutexName := "Global\\TestMutex_Debug"
	fmt.Printf("🔍 尝试创建互斥体: %s\n", mutexName)

	// 转换为UTF16指针
	mutexNamePtr, err := syscall.UTF16PtrFromString(mutexName)
	if err != nil {
		log.Fatalf("❌ 转换互斥体名称失败: %v", err)
	}

	// 调用CreateMutex API
	ret, _, _ := procCreateMutex.Call(
		0,                                     // 安全属性
		0,                                     // 初始所有者
		uintptr(unsafe.Pointer(mutexNamePtr)), // 互斥体名称
	)

	if ret == 0 {
		log.Fatalf("❌ CreateMutex API调用失败")
	}

	// 立即获取错误码
	lastError, _, _ := procGetLastError.Call()
	handle := syscall.Handle(ret)

	fmt.Printf("📝 API调用结果:\n")
	fmt.Printf("   - 句柄: %v\n", handle)
	fmt.Printf("   - 错误码: %d\n", lastError)
	fmt.Printf("   - ERROR_ALREADY_EXISTS: %d\n", ERROR_ALREADY_EXISTS)
	fmt.Printf("   - 错误码是否相等: %t\n", lastError == ERROR_ALREADY_EXISTS)

	if lastError == ERROR_ALREADY_EXISTS {
		fmt.Println("⚠️ 互斥体已存在，这是第二个实例")
		fmt.Println("🔄 程序将在5秒后退出")
		time.Sleep(5 * time.Second)
	} else {
		fmt.Println("✅ 成功创建互斥体，这是首个实例")
		fmt.Println("💡 现在启动第二个实例来测试")
		fmt.Println("⏰ 程序将运行30秒...")

		time.Sleep(30 * time.Second)
		fmt.Println("🔚 首个实例退出")
	}

	// 清理
	procCloseHandle.Call(uintptr(handle))
	fmt.Println("🧹 资源已清理")
}
