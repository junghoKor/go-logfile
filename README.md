# Go High-Performance Safe Logger

**Go Safe Logger**는 윈도우(Windows) 및 리눅스 환경에서 \*\*동시성(Concurrency)\*\*과 \*\*데이터 무결성(Integrity)\*\*을 보장하기 위해 설계된 고성능 비동기 로깅 라이브러리입니다.

단순한 텍스트 기록을 넘어, **파일 잠금(Locking), 자동 로테이션(Rotation), 자동 삭제(Retention), 장애 복구(Retry)** 기능을 내장하여 엔터프라이즈급 안정성을 제공합니다.

## 🚀 주요 기능 (Key Features)

1.  **비동기 및 스레드 안전 (Async & Thread-Safe)**

      * Go 채널(Channel)과 고루틴(Goroutine)을 기반으로 설계되어, 수백 개의 고루틴이 동시에 로그를 남겨도 메인 로직의 성능 저하가 없습니다.
      * 로그 순서가 보장되며, 데이터 경합(Race Condition)이 발생하지 않습니다.

2.  **강력한 파일 잠금 (Windows File Locking)**

      * 윈도우 환경에서 `O_WRONLY` 모드로 파일을 열어, 프로그램 실행 중 **다른 프로세스(메모장 등)가 파일을 수정하거나 훼손하는 것을 운영체제 레벨에서 차단**합니다. (읽기는 가능)

3.  **일자별 자동 로테이션 (Daily Log Rotation)**

      * 프로그램 실행 중 자정(00:00)이 지나면 자동으로 새로운 날짜의 파일(`AppLog_YYYYMMDD.txt`)을 생성하고 전환합니다. 서비스 중단이 없습니다.

4.  **자동 보존 관리 (Retention Policy)**

      * 설정된 기간(예: 60일)이 지난 오래된 로그 파일들을 프로그램 시작 시점과 날짜 변경 시점에 자동으로 삭제하여 디스크 용량을 관리합니다.

5.  **Fail-Fast & Auto-Retry 전략**

      * **초기화 시:** 로그 폴더 권한 없음 등 치명적 오류 발생 시 즉시 `Panic`으로 종료하여, 로그 없이 서비스가 도는 것을 방지합니다.
      * **실행 중:** 파일 교체 실패 등의 오류 발생 시, 서비스를 죽이지 않고 에러를 콘솔에 출력하며 다음 로그 때 재시도를 수행합니다.

6.  **데이터 안전성 (Crash Safety)**

      * `bufio`를 사용해 고속으로 기록하되, **2초 주기 자동 Flush**와 **Panic Recover** 루틴을 통해 프로그램 비정상 종료 시에도 최대 2초 전 데이터 유실을 방지합니다.

-----

## 📦 설치 및 파일 구조 (Installation)

이 라이브러리는 외부 의존성이 없습니다. `logfile.go` 파일만 프로젝트에 포함하면 됩니다.

```bash
MyProject/
├── main.go       # 비즈니스 로직
└── logfile.go    # 로그 시스템 코어 (여기에 포함)
```

-----

## 📖 사용 방법 (Usage)

### 1\. 설정 (Configuration)

`logfile.go`의 `init()` 함수에서 로그 경로, 파일 접두어, 보존 기간을 설정합니다.

```go
// logfile.go
func init() {
    // 경로: c:\log
    // AppLog: 60일 보관
    // CommLog: 30일 보관
    AppLog, CloseAppLog = newLogFunc(`c:\log`, "AppLog", 60)
    CommLog, CloseCommLog = newLogFunc(`c:\log`, "CommLog", 30)
}
```

### 2\. 기본 사용법 (Basic Usage)

`main.go`에서 별도의 초기화 없이 전역 변수처럼 바로 사용합니다. **종료 시 `defer` 처리**만 잊지 마세요.

```go
package main

func main() {
    // [필수] 프로그램 종료 시 안전하게 파일 닫기
    defer CloseAllLogs()

    // 일반 로그 기록
    AppLog("서버가 시작되었습니다. 포트: %d", 8080)

    // 통신 로그 기록
    CommLog("[SEND] 패킷 전송: STX 01 ETX")
}
```

### 3\. 멀티 스레드 환경 (Concurrency)

여러 고루틴에서 동시에 호출해도 안전합니다.

```go
for i := 0; i < 10; i++ {
    go func(id int) {
        // 동시 호출 OK
        AppLog("Worker %d 작업 중...", id)
    }(i)
}
```

-----

## ⚙️ 기술적 상세 (Technical Details)

### 파일명 규칙

  * **포맷:** `{Prefix}_{YYYYMMDD}.txt`
  * **예시:** `AppLog_20251212.txt`
  * **동작:** 같은 날짜에 프로그램을 재시작하면 기존 파일 뒤에 이어서 기록(Append)합니다.

### 안전장치 (Safety Mechanisms)

| 상황 | 동작 방식 |
| :--- | :--- |
| **권한 없음 (시작 시)** | `Panic` 발생, 프로그램 시작 불가 (Fail-Fast) |
| **디스크 꽉 참 (실행 중)** | 콘솔에 에러 출력 후, 다음 로그에서 재시도 (Retry) |
| **프로그램 강제 종료** | 2초 주기로 `Flush` 되므로 최대 2초 데이터만 손실 |
| **런타임 패닉 (Crash)** | `recover` 블록이 작동하여 메모리 버퍼를 파일에 쓰고 죽음 |

### 성능 (Performance)

  * **Buffered I/O:** `bufio.Writer`를 사용하여 시스템 콜(System Call) 횟수를 최소화했습니다.
  * **Non-Blocking:** 채널 버퍼(1000개)가 있어 디스크 쓰기 속도가 느려져도 메인 로직은 블로킹되지 않습니다.

-----

## 📝 라이선스 (License)

This project is licensed under the MIT License.
