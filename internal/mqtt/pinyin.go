package mqtt

import (
	"regexp"
	"strings"

	"github.com/mozillazg/go-pinyin"
)

// 提取设备名中的数字
var numberRegex = regexp.MustCompile(`(\d+)`)

// 将中文字符串转换为拼音首字母缩写
func GetPinyinInitials(str string) string {
	// 使用库转换
	result := pinyin.LazyConvert(str, nil)

	var initials strings.Builder
	for _, py := range result {
		if len(py) > 0 {
			initials.WriteString(strings.ToUpper(string(py[0])))
		}
	}
	return initials.String()
}

// 将设备名转换为缩写（如 摇床01 -> YC01）
func ConvertDeviceName(device string) string {
	// 提取数字
	numbers := numberRegex.FindAllString(device, -1)

	// 提取非数字部分并转换为首字母
	chinesePart := numberRegex.ReplaceAllString(device, "")
	pinyin := GetPinyinInitials(chinesePart)

	// 组合：拼音首字母 + 数字
	result := pinyin
	for _, num := range numbers {
		result += num
	}

	return result
}

// 解析变量名，返回（工段缩写, 设备缩写, 属性缩写）
func ParseVariableName(name string) (string, string, string) {
	parts := strings.Split(name, "_")
	if len(parts) < 3 {
		return GetPinyinInitials(name), ConvertDeviceName(name), GetPinyinInitials(name)
	}

	// 第一部分：工段 -> 拼音首字母
	segment := parts[0]

	// 第二部分：设备 -> 转换为缩写
	device := ConvertDeviceName(parts[1])

	// 第三部分及以后：属性 -> 拼音首字母
	attrs := strings.Join(parts[2:], "")
	attr := GetPinyinInitials(attrs)

	return GetPinyinInitials(segment), device, attr
}
