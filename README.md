我是光年实验室高级招聘经理。
我在github上访问了你的开源项目，你的代码超赞。你最近有没有在看工作机会，我们在招软件开发工程师，拉钩和BOSS等招聘网站也发布了相关岗位，有公司和职位的详细信息。
我们公司在杭州，业务主要做流量增长，是很多大型互联网公司的流量顾问。公司弹性工作制，福利齐全，发展潜力大，良好的办公环境和学习氛围。
公司官网是http://www.gnlab.com,公司地址是杭州市西湖区古墩路紫金广场B座，若你感兴趣，欢迎与我联系，
电话是0571-88839161，手机号：18668131388，微信号：echo 'bGhsaGxoMTEyNAo='|base64 -D ,静待佳音。如有打扰，还请见谅，祝生活愉快工作顺利。

# unparam

	go get mvdan.cc/unparam

Reports unused function parameters and results in your code.

To minimise false positives, it ignores certain cases such as:

* Exported functions (by default, see `-exported`)
* Unnamed and underscore parameters
* Funcs that may satisfy an interface
* Funcs that may satisfy a function signature
* Funcs that are stubs (empty, only error, immediately return, etc)
* Funcs that have multiple implementations via build tags

It also reports results that always return the same value, parameters
that always receive the same value, and results that are never used. In
the last two cases, a minimum number of calls is required to ensure that
the warnings are useful.

False positives can still occur by design. The aim of the tool is to be
as precise as possible - if you find any mistakes, file a bug.
