package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// =========================================================
// 1. 전역 변수 (어디서든 호출 가능)
// =========================================================

var AppLog func(format string, v ...interface{})
var CloseAppLog func()

var CommLog func(format string, v ...interface{})
var CloseCommLog func()

// =========================================================
// 2. 자동 초기화 (Fail Fast 전략)
// =========================================================
func init() {
	// 설정: 경로, 접두어, 보존기간(일)
	// 초기화 실패(권한 없음 등) 시 프로그램은 즉시 Panic으로 종료됩니다.
	AppLog, CloseAppLog = newLogFunc(`c:\log`, "AppLog", 60)   // 60일 보관
	CommLog, CloseCommLog = newLogFunc(`c:\log`, "CommLog", 30) // 30일 보관
}

// 메인 종료 시 호출할 헬퍼
func CloseAllLogs() {
	if CloseCommLog != nil { CloseCommLog() }
	if CloseAppLog != nil { CloseAppLog() }
}

// =========================================================
// 3. 내부 구현 (Internal)
// =========================================================

type internalLogger struct {
	msgChan       chan string
	wg            sync.WaitGroup
	file          *os.File
	writer        *bufio.Writer
	dirPath       string
	filePrefix    string
	currentDate   string
	retentionDays int
}

func newLogFunc(dirPath string, filePrefix string, retentionDays int) (func(string, ...interface{}), func()) {
	// [Fail Fast] 폴더 생성 실패 시 즉시 종료
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		panic(fmt.Sprintf("❌ [LogInit] 폴더 생성 불가: %v", err))
	}

	logger := &internalLogger{
		msgChan:       make(chan string, 1000),
		dirPath:       dirPath,
		filePrefix:    filePrefix,
		retentionDays: retentionDays,
	}

	// [Cleanup] 시작 시 오래된 로그 정리
	logger.cleanOldLogs()

	// [Fail Fast] 초기 파일 열기 실패 시 즉시 종료
	if err := logger.openFile(time.Now()); err != nil {
		panic(fmt.Sprintf("❌ [LogInit] 파일 생성 불가: %v", err))
	}

	logger.wg.Add(1)
	go logger.runWorker()

	fmt.Printf("✅ [System] 로거 가동: %s (보존: %d일)\n", filePrefix, retentionDays)

	// 기록 함수 (비동기 채널 전송)
	logFn := func(format string, v ...interface{}) {
		logger.msgChan <- fmt.Sprintf(format, v...)
	}

	// 종료 함수
	closeFn := func() {
		close(logger.msgChan)
		logger.wg.Wait()
		if logger.writer != nil { logger.writer.Flush() }
		if logger.file != nil { logger.file.Close() }
		fmt.Printf("✅ [System] %s 종료.\n", filePrefix)
	}

	return logFn, closeFn
}

// 파일 열기 (윈도우 쓰기 잠금 포함)
func (l *internalLogger) openFile(t time.Time) error {
	_ = os.MkdirAll(l.dirPath, 0755) // 방어적 수행

	dateStr := t.Format("20060102") // YYYYMMDD
	fileName := fmt.Sprintf("%s_%s.txt", l.filePrefix, dateStr)
	fullPath := filepath.Join(l.dirPath, fileName)

	// O_WRONLY로 열어서 윈도우에서 다른 프로세스의 쓰기를 차단
	f, err := os.OpenFile(fullPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	l.file = f
	l.writer = bufio.NewWriter(f)
	l.currentDate = dateStr
	return nil
}

// 오래된 로그 삭제
func (l *internalLogger) cleanOldLogs() {
	if l.retentionDays <= 0 { return }

	cutoffStr := time.Now().AddDate(0, 0, -l.retentionDays).Format("20060102")

	files, err := os.ReadDir(l.dirPath)
	if err != nil { return }

	for _, file := range files {
		if file.IsDir() { continue }
		name := file.Name()

		// 파일명 검증 (Prefix_YYYYMMDD.txt)
		if !strings.HasPrefix(name, l.filePrefix+"_") || !strings.HasSuffix(name, ".txt") {
			continue
		}
		
		// 날짜 추출
		prefixLen := len(l.filePrefix) + 1
		if len(name) < prefixLen+8+4 { continue }
		fileDateStr := name[prefixLen : prefixLen+8]

		// 문자열 비교로 삭제 여부 결정
		if fileDateStr < cutoffStr {
			fullPath := filepath.Join(l.dirPath, name)
			_ = os.Remove(fullPath)
			fmt.Printf("🗑️ [LogClean] 만료 로그 삭제: %s\n", name)
		}
	}
}

// 백그라운드 워커 (핵심 로직)
func (l *internalLogger) runWorker() {
	defer l.wg.Done()
	ticker := time.NewTicker(2 * time.Second) // 2초 주기 Flush
	defer ticker.Stop()

	// 런타임 패닉 복구
	defer func() {
		if r := recover(); r != nil {
			if l.writer != nil { l.writer.Flush() }
			if l.file != nil { l.file.Sync() }
		}
	}()

	for {
		select {
		case msg, ok := <-l.msgChan:
			if !ok { return } // 채널 닫힘 -> 종료

			now := time.Now()
			today := now.Format("20060102")

			// [Rotation & Retry 로직]
			// 날짜가 바뀌었거나, 이전에 파일 열기에 실패해서 파일이 없는 경우
			if l.file == nil || today != l.currentDate {
				// 기존 파일 정리
				if l.writer != nil { l.writer.Flush() }
				if l.file != nil { l.file.Close() }
				l.file = nil

				// 새 파일 열기 시도
				if err := l.openFile(now); err != nil {
					// 실패 시 죽지 않고 콘솔에 경고 후 재시도(다음 루프)
					fmt.Printf("🔥 [LogSystem] 파일 열기 실패 (재시도 예정): %v\n", err)
					fmt.Println(">> UNSAVED LOG:", msg)
					continue 
				}
				
				fmt.Printf("📅 [LogSystem] 날짜 변경/파일 오픈: %s\n", l.currentDate)
				// 날짜 변경 시 오래된 로그 청소 (비동기)
				go l.cleanOldLogs()
			}

			// 로그 기록 (밀리초 제거됨)
			timestamp := now.Format("2006-01-02 15:04:05")
			l.writer.WriteString(fmt.Sprintf("[%s] %s\n", timestamp, msg))

		case <-ticker.C:
			if l.writer != nil && l.writer.Buffered() > 0 {
				l.writer.Flush()
			}
		}
	}
}