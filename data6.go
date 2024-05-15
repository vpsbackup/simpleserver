package main

var IPMap6 map[string]IPGeoInfo

func init() {
	IPMap6 = make(map[string]IPGeoInfo)
	IPMap6["[2408:873c:9011::55]:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "安徽",
		Version:  "6",
		District: "华东",
	}
	IPMap6["[2409:8c20:8ab1:47:3e::2d]:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "安徽",
		Version:  "6",
		District: "华东",
	}
	IPMap6["[240e:978:a0d:6:3e::4]:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "安徽",
		Version:  "6",
		District: "华东",
	}
	IPMap6["[2409:8c28:6c07:26:3a::c]:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "浙江",
		Version:  "6",
		District: "华东",
	}
	IPMap6["[2408:8719:2000:1:40::34]:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "浙江",
		Version:  "6",
		District: "华东",
	}
	IPMap6["[240e:f7:e000:605:6c::b5]:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "浙江",
		Version:  "6",
		District: "华东",
	}
	IPMap6["[240e:935:a00:10e:40::10]:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "甘肃",
		Version:  "6",
		District: "西北",
	}
	IPMap6["[2408:8776:1:69:70::4]:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "甘肃",
		Version:  "6",
		District: "西北",
	}
	IPMap6["[2409:8c74:f100:e04:60::15]:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "甘肃",
		Version:  "6",
		District: "西北",
	}
	IPMap6["[2409:8c7a:c200:210:5e::d]:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "宁夏",
		Version:  "6",
		District: "西北",
	}
	IPMap6["[2408:8770:0:82:3c::3]:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "宁夏",
		Version:  "6",
		District: "西北",
	}
	IPMap6["[240e:bf:b800:201:3e::a]:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "宁夏",
		Version:  "6",
		District: "西北",
	}
	IPMap6["[240e:bf:c800:2915::59]:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "陕西",
		Version:  "6",
		District: "西北",
	}
	IPMap6["[2408:8670:9cf0:0:45::10]:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "陕西",
		Version:  "6",
		District: "西北",
	}
	IPMap6["[2409:8c74:f100:e04:60::1c]:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "陕西",
		Version:  "6",
		District: "西北",
	}
	IPMap6["[240e:90d:1101:450a:67::24]:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "黑龙江",
		Version:  "6",
		District: "东北",
	}
	IPMap6["[2409:8c1c:300:1b:5e::30]:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "黑龙江",
		Version:  "6",
		District: "东北",
	}
	IPMap6["[2408:872f:20:210::13e]:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "黑龙江",
		Version:  "6",
		District: "东北",
	}
	IPMap6["[2408:8726:7000:800c:60::24]:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "内蒙古",
		Version:  "6",
		District: "华北",
	}
	IPMap6["[2409:8c0c:310:4004:60::d]:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "内蒙古",
		Version:  "6",
		District: "华北",
	}
	IPMap6["[240e:928:501:23:39::17]:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "内蒙古",
		Version:  "6",
		District: "华北",
	}
	IPMap6["[2409:8c6a:b021:50::62]:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "贵州",
		Version:  "6",
		District: "西南",
	}
	IPMap6["[2408:876a:1000:e2:4c::]:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "贵州",
		Version:  "6",
		District: "西南",
	}
	IPMap6["[240e:938:a05:32:3d::]:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "贵州",
		Version:  "6",
		District: "西南",
	}
	IPMap6["[2408:8763:0:221:3a::1b]:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "云南",
		Version:  "6",
		District: "西南",
	}
	IPMap6["[2409:8c6a:b021:24::1a]:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "云南",
		Version:  "6",
		District: "西南",
	}
	IPMap6["[240e:94c:0:115:60::3]:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "云南",
		Version:  "6",
		District: "西南",
	}
	IPMap6["[2408:8776:1:69:70::4]:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "新疆",
		Version:  "6",
		District: "西北",
	}
	IPMap6["[240e:935:a00:115::278]:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "新疆",
		Version:  "6",
		District: "西北",
	}
	IPMap6["[2409:8c74:f100:e04:60::17]:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "新疆",
		Version:  "6",
		District: "西北",
	}
	IPMap6["[240e:bf:c800:2300:1f::]:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "青海",
		Version:  "6",
		District: "西北",
	}
	IPMap6["[2409:8c78:100:21:3a::16]:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "青海",
		Version:  "6",
		District: "西北",
	}
	IPMap6["[2408:8776:1:69:70::4]:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "青海",
		Version:  "6",
		District: "西北",
	}
	IPMap6["[240e:97d:4:1e02:13::]:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "广东",
		Version:  "6",
		District: "华南",
	}
	IPMap6["[2408:874d:a00:b::53]:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "广东",
		Version:  "6",
		District: "华南",
	}
	IPMap6["[2408:8670:3af0:34:40::13]:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "广东",
		Version:  "6",
		District: "华南",
	}
	IPMap6["[2408:8763:0:221:3a::1b]:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "四川",
		Version:  "6",
		District: "西南",
	}
	IPMap6["[2409:8c74:f100:e04:60::1c]:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "四川",
		Version:  "6",
		District: "西南",
	}
	IPMap6["[240e:974:c900:1:70::10]:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "四川",
		Version:  "6",
		District: "西南",
	}
	IPMap6["[2409:875c:7f8:30:64::75]:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "湖南",
		Version:  "6",
		District: "华中",
	}
	IPMap6["[240e:c2:1800:31::1af]:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "湖南",
		Version:  "6",
		District: "华中",
	}
	IPMap6["[2408:872b:e02:101:6c::143]:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "湖南",
		Version:  "6",
		District: "华中",
	}
	IPMap6["[2408:8726:1001:151:62::45]:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "山西",
		Version:  "6",
		District: "华北",
	}
	IPMap6["[240e:944:e:5:63::ad]:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "山西",
		Version:  "6",
		District: "华北",
	}
	IPMap6["[2409:8c0c:310:21a::99]:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "山西",
		Version:  "6",
		District: "华北",
	}
	IPMap6["[240e:974:c900:3:2a::]:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "西藏",
		Version:  "6",
		District: "西南",
	}
	IPMap6["[2409:8c6c:561:8110:38::29]:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "西藏",
		Version:  "6",
		District: "西南",
	}
	IPMap6["[2408:876c:1700:142:70::36]:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "西藏",
		Version:  "6",
		District: "西南",
	}
	IPMap6["[240e:96c:6400:702:30::1e]:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "上海",
		Version:  "6",
		District: "华东",
	}
	IPMap6["[2408:8740:71fc:410::2f]:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "上海",
		Version:  "6",
		District: "华东",
	}
	IPMap6["[2408:8670:3af0:32:40::1]:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "上海",
		Version:  "6",
		District: "华东",
	}
	IPMap6["[2408:874f:b000:4:253::71]:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "湖北",
		Version:  "6",
		District: "华中",
	}
	IPMap6["[240e:f7:4d0f:101:70::24]:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "湖北",
		Version:  "6",
		District: "华中",
	}
	IPMap6["[2409:8c0c:310:204::5f]:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "湖北",
		Version:  "6",
		District: "华中",
	}
	IPMap6["[2409:8c44:2:10a:68::2b]:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "河南",
		Version:  "6",
		District: "华中",
	}
	IPMap6["[240e:944:e:5:63::ae]:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "河南",
		Version:  "6",
		District: "华中",
	}
	IPMap6["[2408:8722:3801:10:6c::26]:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "河南",
		Version:  "6",
		District: "华中",
	}
	IPMap6["[2409:8c28:6c07:26:3a::f]:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "北京",
		Version:  "6",
		District: "华北",
	}
	IPMap6["[2408:8719:2000:1c0:6c::3d]:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "北京",
		Version:  "6",
		District: "华北",
	}
	IPMap6["[240e:964:1100:111::46]:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "北京",
		Version:  "6",
		District: "华北",
	}
	IPMap6["[240e:90c:a201:0:67::9a]:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "辽宁",
		Version:  "6",
		District: "东北",
	}
	IPMap6["[2409:8c1c:300:17:5e::4]:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "辽宁",
		Version:  "6",
		District: "东北",
	}
	IPMap6["[2408:872f:20:210::13b]:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "辽宁",
		Version:  "6",
		District: "东北",
	}
	IPMap6["[2408:8719:2000:1c0:6c::3d]:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "山东",
		Version:  "6",
		District: "华东",
	}
	IPMap6["[240e:946:6004:3:32::1a]:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "山东",
		Version:  "6",
		District: "华东",
	}
	IPMap6["[2409:8c3c:8100:10:5e::8]:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "山东",
		Version:  "6",
		District: "华东",
	}
	IPMap6["[2409:801e:300e:115:37::2]:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "江西",
		Version:  "6",
		District: "华东",
	}
	IPMap6["[240e:cd:ff00:113:70::16]:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "江西",
		Version:  "6",
		District: "华东",
	}
	IPMap6["[2408:874d:a00:b::53]:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "江西",
		Version:  "6",
		District: "华东",
	}
	IPMap6["[2408:8749:c110:804:70::27]:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "江苏",
		Version:  "6",
		District: "华东",
	}
	IPMap6["[2409:801e:300e:115:37::2]:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "江苏",
		Version:  "6",
		District: "华东",
	}
	IPMap6["[2408:873c:3011:10::5d]:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "江苏",
		Version:  "6",
		District: "华东",
	}
	IPMap6["[2409:8c54:2800:9111:45::1b]:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "福建",
		Version:  "6",
		District: "华东",
	}
	IPMap6["[2408:8749:c110:701:3c::1a]:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "福建",
		Version:  "6",
		District: "华东",
	}
	IPMap6["[240e:914:3005:1::52]:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "福建",
		Version:  "6",
		District: "华东",
	}
	IPMap6["[240e:950:1:2002:67::]:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "广西",
		Version:  "6",
		District: "华南",
	}
	IPMap6["[2408:8740:71fc:410::30]:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "广西",
		Version:  "6",
		District: "华南",
	}
	IPMap6["[2409:875e:5088:c1:37::2f]:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "广西",
		Version:  "6",
		District: "华南",
	}
	IPMap6["[2408:8726:1001:151:62::45]:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "重庆",
		Version:  "6",
		District: "西南",
	}
	IPMap6["[240e:94c:4000:1611:31::a]:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "重庆",
		Version:  "6",
		District: "西南",
	}
	IPMap6["[2409:8760:1e81:51:40::18]:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "重庆",
		Version:  "6",
		District: "西南",
	}
	IPMap6["[2408:8719:2000:1c0:6c::3e]:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "天津",
		Version:  "6",
		District: "华北",
	}
	IPMap6["[2409:8c02:218:102:42::8]:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "天津",
		Version:  "6",
		District: "华北",
	}
	IPMap6["[240e:928:501:23:39::13]:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "天津",
		Version:  "6",
		District: "华北",
	}
	IPMap6["[2408:871a:5500:c:20::3]:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "河北",
		Version:  "6",
		District: "华北",
	}
	IPMap6["[2409:8c0c:310:4004:60::14]:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "河北",
		Version:  "6",
		District: "华北",
	}
	IPMap6["[240e:944:e:5:63::ad]:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "河北",
		Version:  "6",
		District: "华北",
	}
	IPMap6["[2409:8c5e:5000:210:79::d]:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "海南",
		Version:  "6",
		District: "华南",
	}
	IPMap6["[2408:8760:112:100::52]:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "海南",
		Version:  "6",
		District: "华南",
	}
	IPMap6["[240e:914:a003:3:70::18]:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "海南",
		Version:  "6",
		District: "华南",
	}
	IPMap6["[2409:8c1c:300:1b:5e::4f]:80"] = IPGeoInfo{
		ISP:      "移动",
		Region:   "吉林",
		Version:  "6",
		District: "东北",
	}
	IPMap6["[240e:90d:1101:4102:3d::8]:80"] = IPGeoInfo{
		ISP:      "电信",
		Region:   "吉林",
		Version:  "6",
		District: "东北",
	}
	IPMap6["[2408:872f:20:210::13a]:80"] = IPGeoInfo{
		ISP:      "联通",
		Region:   "吉林",
		Version:  "6",
		District: "东北",
	}
}
