package main

type IPGeoInfo struct {
	ISP      string
	Region   string
	Version  string
	District string
}

var IPMap map[string]IPGeoInfo

func init() {
	IPMap = make(map[string]IPGeoInfo)
	IPMap["59.81.65.42:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "上海",
		Version:  "4",
		District: "华东",
	}
	IPMap["183.194.219.220:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "上海",
		Version:  "4",
		District: "华东",
	}
	IPMap["101.227.191.14:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "上海",
		Version:  "4",
		District: "华东",
	}
	IPMap["61.182.138.156:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "河北",
		Version:  "4",
		District: "华北",
	}
	IPMap["111.62.229.100:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "河北",
		Version:  "4",
		District: "华北",
	}
	IPMap["27.185.242.215:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "河北",
		Version:  "4",
		District: "华北",
	}
	IPMap["60.221.18.41:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "山西",
		Version:  "4",
		District: "华北",
	}
	IPMap["183.201.244.91:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "山西",
		Version:  "4",
		District: "华北",
	}
	IPMap["1.71.157.41:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "山西",
		Version:  "4",
		District: "华北",
	}
	IPMap["218.61.211.132:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "辽宁",
		Version:  "4",
		District: "东北",
	}
	IPMap["36.131.156.145:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "辽宁",
		Version:  "4",
		District: "东北",
	}
	IPMap["123.184.58.41:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "辽宁",
		Version:  "4",
		District: "东北",
	}
	IPMap["122.143.8.41:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "吉林",
		Version:  "4",
		District: "东北",
	}
	IPMap["111.27.127.176:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "吉林",
		Version:  "4",
		District: "东北",
	}
	IPMap["123.172.127.217:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "吉林",
		Version:  "4",
		District: "东北",
	}
	IPMap["113.7.211.140:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "黑龙江",
		Version:  "4",
		District: "东北",
	}
	IPMap["111.42.190.25:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "黑龙江",
		Version:  "4",
		District: "东北",
	}
	IPMap["42.101.84.132:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "黑龙江",
		Version:  "4",
		District: "东北",
	}
	IPMap["122.96.235.165:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "江苏",
		Version:  "4",
		District: "华东",
	}
	IPMap["36.156.92.132:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "江苏",
		Version:  "4",
		District: "华东",
	}
	IPMap["58.215.210.220:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "江苏",
		Version:  "4",
		District: "华东",
	}
	IPMap["101.69.194.224:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "浙江",
		Version:  "4",
		District: "华东",
	}
	IPMap["117.147.213.41:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "浙江",
		Version:  "4",
		District: "华东",
	}
	IPMap["115.220.14.91:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "浙江",
		Version:  "4",
		District: "华东",
	}
	IPMap["112.132.208.41:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "安徽",
		Version:  "4",
		District: "华东",
	}
	IPMap["112.29.198.100:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "安徽",
		Version:  "4",
		District: "华东",
	}
	IPMap["223.247.108.251:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "安徽",
		Version:  "4",
		District: "华东",
	}
	IPMap["36.248.48.139:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "福建",
		Version:  "4",
		District: "华东",
	}
	IPMap["112.50.96.88:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "福建",
		Version:  "4",
		District: "华东",
	}
	IPMap["106.126.10.28:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "福建",
		Version:  "4",
		District: "华东",
	}
	IPMap["116.153.69.224:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "江西",
		Version:  "4",
		District: "华东",
	}
	IPMap["117.168.150.249:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "江西",
		Version:  "4",
		District: "华东",
	}
	IPMap["106.227.22.132:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "江西",
		Version:  "4",
		District: "华东",
	}
	IPMap["112.240.56.143:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "山东",
		Version:  "4",
		District: "华东",
	}
	IPMap["120.220.145.91:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "山东",
		Version:  "4",
		District: "华东",
	}
	IPMap["144.123.160.140:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "山东",
		Version:  "4",
		District: "华东",
	}
	IPMap["123.6.65.101:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "河南",
		Version:  "4",
		District: "华中",
	}
	IPMap["111.7.99.220:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "河南",
		Version:  "4",
		District: "华中",
	}
	IPMap["171.15.110.220:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "河南",
		Version:  "4",
		District: "华中",
	}
	IPMap["122.189.226.138:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "湖北",
		Version:  "4",
		District: "华中",
	}
	IPMap["111.47.131.101:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "湖北",
		Version:  "4",
		District: "华中",
	}
	IPMap["111.170.8.60:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "湖北",
		Version:  "4",
		District: "华中",
	}
	IPMap["116.162.28.220:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "湖南",
		Version:  "4",
		District: "华中",
	}
	IPMap["120.226.192.91:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "湖南",
		Version:  "4",
		District: "华中",
	}
	IPMap["113.240.117.108:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "湖南",
		Version:  "4",
		District: "华中",
	}
	IPMap["112.90.211.100:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "广东",
		Version:  "4",
		District: "华南",
	}
	IPMap["183.240.65.191:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "广东",
		Version:  "4",
		District: "华南",
	}
	IPMap["183.36.23.111:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "广东",
		Version:  "4",
		District: "华南",
	}
	IPMap["153.0.226.35:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "海南",
		Version:  "4",
		District: "华南",
	}
	IPMap["111.29.29.219:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "海南",
		Version:  "4",
		District: "华南",
	}
	IPMap["124.225.43.220:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "海南",
		Version:  "4",
		District: "华南",
	}
	IPMap["101.206.163.49:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "四川",
		Version:  "4",
		District: "西南",
	}
	IPMap["183.220.151.41:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "四川",
		Version:  "4",
		District: "西南",
	}
	IPMap["118.123.218.220:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "四川",
		Version:  "4",
		District: "西南",
	}
	IPMap["117.187.254.132:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "贵州",
		Version:  "4",
		District: "西南",
	}
	IPMap["61.243.18.220:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "贵州",
		Version:  "4",
		District: "西南",
	}
	IPMap["58.42.61.132:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "贵州",
		Version:  "4",
		District: "西南",
	}
	IPMap["14.204.150.41:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "云南",
		Version:  "4",
		District: "西南",
	}
	IPMap["36.147.44.219:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "云南",
		Version:  "4",
		District: "西南",
	}
	IPMap["222.221.102.220:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "云南",
		Version:  "4",
		District: "西南",
	}
	IPMap["123.139.127.132:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "陕西",
		Version:  "4",
		District: "西北",
	}
	IPMap["111.19.148.100:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "陕西",
		Version:  "4",
		District: "西北",
	}
	IPMap["124.115.14.100:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "陕西",
		Version:  "4",
		District: "西北",
	}
	IPMap["59.81.94.53:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "甘肃",
		Version:  "4",
		District: "西北",
	}
	IPMap["117.157.16.41:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "甘肃",
		Version:  "4",
		District: "西北",
	}
	IPMap["118.182.228.91:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "甘肃",
		Version:  "4",
		District: "西北",
	}
	IPMap["116.177.237.137:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "青海",
		Version:  "4",
		District: "西北",
	}
	IPMap["111.12.152.170:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "青海",
		Version:  "4",
		District: "西北",
	}
	IPMap["223.221.216.219:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "青海",
		Version:  "4",
		District: "西北",
	}
	IPMap["116.114.98.41:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "内蒙古",
		Version:  "4",
		District: "华北",
	}
	IPMap["117.161.76.41:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "内蒙古",
		Version:  "4",
		District: "华北",
	}
	IPMap["110.76.186.70:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "内蒙古",
		Version:  "4",
		District: "华北",
	}
	IPMap["171.39.5.51:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "广西",
		Version:  "4",
		District: "华南",
	}
	IPMap["36.136.112.41:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "广西",
		Version:  "4",
		District: "华南",
	}
	IPMap["222.217.93.55:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "广西",
		Version:  "4",
		District: "华南",
	}
	IPMap["43.242.165.35:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "西藏",
		Version:  "4",
		District: "西南",
	}
	IPMap["117.180.234.41:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "西藏",
		Version:  "4",
		District: "西南",
	}
	IPMap["113.62.176.89:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "西藏",
		Version:  "4",
		District: "西南",
	}
	IPMap["116.129.226.28:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "宁夏",
		Version:  "4",
		District: "西北",
	}
	IPMap["111.51.155.214:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "宁夏",
		Version:  "4",
		District: "西北",
	}
	IPMap["222.75.44.220:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "宁夏",
		Version:  "4",
		District: "西北",
	}
	IPMap["116.178.77.40:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "新疆",
		Version:  "4",
		District: "西北",
	}
	IPMap["36.189.208.164:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "新疆",
		Version:  "4",
		District: "西北",
	}
	IPMap["110.157.243.45:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "新疆",
		Version:  "4",
		District: "西北",
	}
	IPMap["202.108.29.159:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "北京",
		Version:  "4",
		District: "华北",
	}
	IPMap["222.35.73.1:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "北京",
		Version:  "4",
		District: "华北",
	}
	IPMap["220.181.173.35:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "北京",
		Version:  "4",
		District: "华北",
	}
	IPMap["116.78.119.56:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "天津",
		Version:  "4",
		District: "华北",
	}
	IPMap["111.31.236.35:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "天津",
		Version:  "4",
		District: "华北",
	}
	IPMap["42.81.98.35:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "天津",
		Version:  "4",
		District: "华北",
	}
	IPMap["113.207.69.190:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "重庆",
		Version:  "4",
		District: "西南",
	}
	IPMap["221.178.81.101:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "重庆",
		Version:  "4",
		District: "西南",
	}
	IPMap["119.84.131.101:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "重庆",
		Version:  "4",
		District: "西南",
	}
}
