package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

func main() {
	// [필수] 프로그램 종료 시 로그 정리
	defer CloseAllLogs()

	fmt.Println("🔥 [System] 고부하 멀티스레드 로깅 테스트 시작...")

	// 랜덤 시드 설정 (실행할 때마다 다른 패턴)
	rand.Seed(time.Now().UnixNano())

	var wg sync.WaitGroup

	// =====================================================
	// 시나리오 1: DB 작업 시뮬레이션 (AppLog 위주)
	// - 5개의 워커가 각자 다른 속도로 로그를 남김
	// =====================================================
	fmt.Println("   🚀 [Scenario 1] DB Workers (5 threads)")
	for i := 1; i <= 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			simulateDBWorker(id)
		}(i)
	}

	// =====================================================
	// 시나리오 2: 네트워크 트래픽 시뮬레이션 (CommLog 위주)
	// - 5개의 워커가 매우 빠르게 통신 로그를 남김
	// =====================================================
	fmt.Println("   🚀 [Scenario 2] Network Workers (5 threads)")
	for i := 1; i <= 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			simulateNetworkWorker(id)
		}(i)
	}

	// =====================================================
	// 시나리오 3: 복합 처리 (AppLog + CommLog 동시 사용)
	// - 두 로그 파일을 동시에 건드려도 Deadlock이 없는지 확인
	// =====================================================
	fmt.Println("   🚀 [Scenario 3] Mixed Logic Workers (3 threads)")
	for i := 1; i <= 3; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			simulateMixedLogic(id)
		}(i)
	}

	// 워커들이 도는 동안 잠시 대기...
	time.Sleep(1 * time.Second)

	// =====================================================
	// 시나리오 4: 순간 폭주 (Burst) 테스트
	// - 100개의 고루틴을 '동시에' 띄워서 로그 시스템 부하 테스트
	// =====================================================
	fmt.Println("   💥 [Scenario 4] BURST TEST (100 goroutines at once!)")
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			// 아주 짧은 순간에 로그 기록 시도
			AppLog("[BURST] 폭주 테스트 #%d - 시스템 살아있나?", idx)
		}(i)
	}

	// 모든 고루틴 종료 대기
	wg.Wait()
	
	fmt.Println("   ✅ 모든 워커 작업 완료.")
	fmt.Println("   ⏳ 3초 대기 (남은 로그 Flush)...")
	time.Sleep(3 * time.Second)
	
	fmt.Println("👋 프로그램 종료")
}

// ---------------------------------------------------------
// [Helper] DB 워커: 불규칙한 작업 시간 시뮬레이션
// ---------------------------------------------------------
func simulateDBWorker(id int) {
	for j := 0; j < 5; j++ {
		// 작업 시작 로그
		AppLog("[DB-%02d] 트랜잭션 시작 (Job %d)", id, j)
		
		// 랜덤한 시간만큼 작업 (10ms ~ 100ms)
		duration := time.Duration(rand.Intn(90)+10) * time.Millisecond
		time.Sleep(duration)
		
		// 작업 완료 로그
		AppLog("[DB-%02d] 쿼리 완료 (소요시간: %v)", id, duration)
	}
}

// ---------------------------------------------------------
// [Helper] 네트워크 워커: 빠른 패킷 로그
// ---------------------------------------------------------
func simulateNetworkWorker(id int) {
	for j := 0; j < 10; j++ {
		// 통신 로그 기록
		CommLog("[NET-%02d] SEND Packet seq=%d size=%d bytes", id, j, rand.Intn(1024))
		
		// 아주 짧은 대기 (네트워크 지연)
		time.Sleep(time.Duration(rand.Intn(20)) * time.Millisecond)
		
		CommLog("[NET-%02d] RECV ACK seq=%d", id, j)
	}
}

// ---------------------------------------------------------
// [Helper] 복합 로직: 두 로거를 동시에 호출
// ---------------------------------------------------------
func simulateMixedLogic(id int) {
	for j := 0; j < 3; j++ {
		// 1. 비즈니스 로직 로그
		AppLog("[MIX-%02d] 사용자 요청 처리 중...", id)
		
		// 2. 외부 API 호출 로그 (CommLog)
		CommLog("[API-%02d] GET /user/info", id)
		
		time.Sleep(50 * time.Millisecond)
		
		// 3. 결과 로그
		AppLog("[MIX-%02d] 처리 성공", id)
	}
}