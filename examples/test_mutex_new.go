// Package main 全新的互斥体测试程序
// 使用OpenMutex方法检查互斥体是否存在
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
	MUTEX_ALL_ACCESS     = 0x1F0001 // 互斥体完全访问权限
	ERROR_FILE_NOT_FOUND = 2        // 文件不存在错误码
)

// Windows API 函数声明
var (
	kernel32        = syscall.NewLazyDLL("kernel32.dll") // 加载内核动态链接库
	procCreateMutex = kernel32.NewProc("CreateMutexW")   // 创建互斥体API函数
	procOpenMutex   = kernel32.NewProc("OpenMutexW")     // 打开互斥体API函数
	procCloseHandle = kernel32.NewProc("CloseHandle")    // 关闭句柄API函数
)

func main() {
	fmt.Printf("🧪 全新互斥体测试程序启动 (PID: %d)\n", os.Getpid())

	// 互斥体名称
	mutexName := "Global\\TestMutex_New"
	fmt.Printf("🔍 检查互斥体是否存在: %s\n", mutexName)

	// 转换为UTF16指针
	mutexNamePtr, err := syscall.UTF16PtrFromString(mutexName)
	if err != nil {
		log.Fatalf("❌ 转换互斥体名称失败: %v", err)
	}

	// 首先尝试打开现有的互斥体
	openHandle, _, _ := procOpenMutex.Call(
		MUTEX_ALL_ACCESS,                      // 访问权限
		0,                                     // 不继承句柄
		uintptr(unsafe.Pointer(mutexNamePtr)), // 互斥体名称
	)

	if openHandle != 0 {
		// 互斥体已存在
		fmt.Println("⚠️ 互斥体已存在，这是第二个实例")
		fmt.Println("🔄 程序将在5秒后退出")
		procCloseHandle.Call(uintptr(openHandle))
		time.Sleep(5 * time.Second)
		return
	}

	// 互斥体不存在，创建新的
	fmt.Println("✅ 互斥体不存在，创建新的互斥体")

	createHandle, _, _ := procCreateMutex.Call(
		0,                                     // 安全属性
		1,                                     // 初始拥有者 - 立即获得所有权
		uintptr(unsafe.Pointer(mutexNamePtr)), // 互斥体名称
	)

	if createHandle == 0 {
		log.Fatalf("❌ CreateMutex失败")
	}

	fmt.Printf("📝 成功创建互斥体，句柄: %v\n", createHandle)
	fmt.Println("🚀 这是首个实例，开始运行...")
	fmt.Println("💡 现在启动第二个实例来测试")
	fmt.Println("⏰ 程序将运行30秒...")

	time.Sleep(30 * time.Second)

	// 清理
	procCloseHandle.Call(uintptr(createHandle))
	fmt.Println("🔚 首个实例退出")
	fmt.Println("🧹 资源已清理")
}
