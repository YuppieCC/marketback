package main

import (
	"fmt"
	"marketcontrol/pkg/utils"
)

func main() {
	// 测试用例0：正常分割
	testCase0 := struct {
		n      int
		min    int
		max    int
		maxLen int
		expect string
	}{
		n:      11,
		min:    2,
		max:    5,
		maxLen: 3,
		expect: "应该分割成不超过3个组",
	}
	result0 := utils.GreedySplit(testCase0.n, testCase0.min, testCase0.max, testCase0.maxLen)
	fmt.Printf("测试用例0 - %s\n", testCase0.expect)
	fmt.Printf("输入: n=%d, min=%d, max=%d, maxLen=%d\n", testCase0.n, testCase0.min, testCase0.max, testCase0.maxLen)
	fmt.Printf("输出: %v\n", result0)
	fmt.Printf("和: %d\n\n", sum(result0))

	// 测试用例1：正常分割
	testCase1 := struct {
		n      int
		min    int
		max    int
		maxLen int
		expect string
	}{
		n:      100,
		min:    10,
		max:    30,
		maxLen: 4,
		expect: "应该分割成不超过4个组",
	}
	result1 := utils.GreedySplit(testCase1.n, testCase1.min, testCase1.max, testCase1.maxLen)
	fmt.Printf("测试用例1 - %s\n", testCase1.expect)
	fmt.Printf("输入: n=%d, min=%d, max=%d, maxLen=%d\n", testCase1.n, testCase1.min, testCase1.max, testCase1.maxLen)
	fmt.Printf("输出: %v\n", result1)
	fmt.Printf("和: %d\n\n", sum(result1))

	// 测试用例2：边界情况
	testCase2 := struct {
		n      int
		min    int
		max    int
		maxLen int
		expect string
	}{
		n:      50,
		min:    20,
		max:    25,
		maxLen: 2,
		expect: "应该分割成不超过2个组",
	}
	result2 := utils.GreedySplit(testCase2.n, testCase2.min, testCase2.max, testCase2.maxLen)
	fmt.Printf("测试用例2 - %s\n", testCase2.expect)
	fmt.Printf("输入: n=%d, min=%d, max=%d, maxLen=%d\n", testCase2.n, testCase2.min, testCase2.max, testCase2.maxLen)
	fmt.Printf("输出: %v\n", result2)
	fmt.Printf("和: %d\n\n", sum(result2))

	// 测试用例3：极端情况
	testCase3 := struct {
		n      int
		min    int
		max    int
		maxLen int
		expect string
	}{
		n:      15,
		min:    20,
		max:    30,
		maxLen: 5,
		expect: "当n小于min时应该返回空数组",
	}
	result3 := utils.GreedySplit(testCase3.n, testCase3.min, testCase3.max, testCase3.maxLen)
	fmt.Printf("测试用例3 - %s\n", testCase3.expect)
	fmt.Printf("输入: n=%d, min=%d, max=%d, maxLen=%d\n", testCase3.n, testCase3.min, testCase3.max, testCase3.maxLen)
	fmt.Printf("输出: %v\n", result3)
	fmt.Printf("和: %d\n\n", sum(result3))
}

// 辅助函数：计算切片中所有数字的和
func sum(numbers []int) int {
	total := 0
	for _, num := range numbers {
		total += num
	}
	return total
}
