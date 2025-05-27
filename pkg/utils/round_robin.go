package utils

import "sync/atomic"

// RoundRobin은 라운드 로빈 로드 밸런싱을 구현합니다
type RoundRobin struct {
	currentIndex uint32
}

// NewRoundRobin은 새로운 RoundRobin 인스턴스를 생성합니다
func NewRoundRobin() *RoundRobin {
	return &RoundRobin{currentIndex: 0}
}

// Next는 주어진 슬라이스에서 라운드 로빈 방식으로 다음 항목을 선택합니다
func (rr *RoundRobin) Next(items []string) string {
	if len(items) == 0 {
		return ""
	}

	// 원자적으로 인덱스 증가
	index := atomic.AddUint32(&rr.currentIndex, 1)
	// 슬라이스 길이로 나머지 연산하여 범위 내 값 유지
	return items[int(index-1)%len(items)]
}

// NextIndex는 주어진 슬라이스 길이에 대해 다음 인덱스를 반환합니다
func (rr *RoundRobin) NextIndex(length int) int {
	if length <= 0 {
		return -1
	}

	index := atomic.AddUint32(&rr.currentIndex, 1)
	return int(index-1) % length
}
