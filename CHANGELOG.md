# 更新日志

## 0.0.6

- 登录态自动续期：QQ 与微信会话过期后自动刷新，续期失败可重试，不再直接退回未登录
- 个人页显示会员到期时间；新增歌单内搜索，可直接在我的歌单中查找歌曲
- 歌词字号可调；恢复主窗口时立即刷新歌词，不再残留旧内容
- 修复 Linux Wayland 下桌面歌词的定位、层级与锁定后鼠标穿透，桌面歌词在 Wayland 会话中可正常使用
- 桌面音频在采样率切换时平滑过渡，退出前淡出，避免切换与关闭时的爆音；每日推荐唱片跟随当前封面旋转
- 任务栏歌词的右键菜单不再溢出窗口，播放控制浮层与队列统一材质；修正「正在播放」控件紧贴分页标签

## 0.0.5

- 降低封面内存占用：同一封面只解码、下载一次，背景与主题色按需派生，展示用的封面不再多解一份图像；缓存按最近使用淘汰，被 Now Playing 持有的封面可原样恢复
- 未缓存的歌曲改用临时文件播放，不再整曲驻留内存，进程退出自动清理
- 升级 MyGo 至 v0.8.3：歌词遮罩与图形资源按需解析，闲置遮罩与临时资源及时回收

## 0.0.4

- 新增桌面歌词颜色自定义：六个预设色与原生取色器，歌词随演唱从白色渐变到所选颜色，可调透明度
- Linux 新增桌面歌词，锁定后点击穿透，面板可直接拖动并记住位置
- 应用图标去除外阴影；macOS 改用带标准边距的专用 ICNS，修正程序坞中的图标尺寸
- 降低播放与歌词的资源占用：频谱按需异步计算、FLAC 解码复用缓存，歌词描边与翻译层复用布局，空闲时段不再持续重绘

## 0.0.4-beta.1

- 预发布测试版：CI 在转正前校验三平台两架构的安装包与便携包是否齐全，并把 Release 资产精简为安装包、便携包与自动更新清单
- 这是 beta 版，正式用户默认不会收到更新

## 0.0.3

- 新增「主题色跟随封面」：开启后按钮、开关、选中态、焦点环和播放栏进度条使用当前歌曲封面的主题色；默认关闭，可在设置 → 外观里开启
- 队列浮层改用与其他弹层一致的玻璃材质，设置里的主题模式文案更清晰
- 修复任务栏歌词双行排版在字体度量差异下的误报，发布前自动运行构建、静态检查与测试
- 本地构建不再生成差分包，差分包只在发版时生成

## 0.0.2

- 应用更名为 Yunhe：窗口标题、主程序、安装包与便携包同步更名；旧版设置、播放队列和缓存自动迁移，登录态保持不变
- GitHub 仓库更名为 `clover1420/yunhe`，更新源与下载地址同步
- 设置中的「外观」「侧边栏材质」从「常规」移到「外观」分类

## 0.0.1

界面按 Cloudy 的样子重做，功能不变：

- 配色从鼠尾草绿的玻璃质感，换成中性灰白与一抹红；圆角、阴影、边界随之统一
- 播放栏改为浮起的玻璃胶囊，边缘弯折背后的内容；通透度与折射强度可调，也可关掉
- 页面切换有了方向感：往里打开从右侧进，返回从左侧进，切换标签则原地更新
- 启动时一枚唱片转上来，再淡入应用
- 正在播放的歌词在封面的颜色里发光，长音的字会轻轻鼓起，强度可调
- 设置改为浮在窗口上的一层玻璃面板，左侧分「常规」「外观」「播放与歌词」「关于」，各自记住滚动位置
- 新增字体选择：列出这台电脑上装了的字体，中文常用字体排在前面
- 移除官网目录，应用不再依赖它构建

> 从这一版起更新签名密钥已更换、版本号重新起算。此前 0.3.0 及更早的安装不会自动更新到本版，需要手动下载安装。

## 0.3.0

- Windows 和 macOS 新增实验性桌面歌词，独立显示逐字高亮的当前歌词，支持拖动、调整宽度和字号、锁定位置及悬停播放控制
- Windows 新增实验性任务栏歌词，在空间允许的横向主任务栏中显示当前歌词与封面；点击封面可打开播放控制浮层，进行上一首、播放/暂停、下一首和进度跳转；灵感来源[TaskbarLyrics](https://github.com/ANYNC/TaskbarLyrics)
- 设置中可统一选择桌面或任务栏歌词，并可从播放栏或托盘快速开关；关闭歌词后会保留当前模式和桌面歌词偏好
- 歌曲菜单新增「下一首播放」，可将歌曲直接插入当前播放项之后

### 下载指南

| 处理器架构 | Windows | macOS | Linux |
| :--- | :--- | :--- | :--- |
| amd64 | [安装包 .exe](https://github.com/Amamiya23/mygo-qyin/releases/download/v0.3.0/lilt-setup-0.3.0-amd64.exe) | [安装包 .dmg](https://github.com/Amamiya23/mygo-qyin/releases/download/v0.3.0/lilt-0.3.0-amd64.dmg) | [.deb](https://github.com/Amamiya23/mygo-qyin/releases/download/v0.3.0/lilt_0.3.0_amd64.deb) · [.tar.gz](https://github.com/Amamiya23/mygo-qyin/releases/download/v0.3.0/lilt-0.3.0-linux-amd64.tar.gz) |
| arm64 | [安装包 .exe](https://github.com/Amamiya23/mygo-qyin/releases/download/v0.3.0/lilt-setup-0.3.0-arm64.exe) | [安装包 .dmg](https://github.com/Amamiya23/mygo-qyin/releases/download/v0.3.0/lilt-0.3.0-arm64.dmg) | [.deb](https://github.com/Amamiya23/mygo-qyin/releases/download/v0.3.0/lilt_0.3.0_arm64.deb) · [.tar.gz](https://github.com/Amamiya23/mygo-qyin/releases/download/v0.3.0/lilt-0.3.0-linux-arm64.tar.gz) |

> Linux：Ubuntu / Debian 等支持 deb 的发行版选 `.deb`，其他发行版选 `.tar.gz`。

## 0.2.2

- 逐字歌词完全适配 QQ 音乐官方 QRC 逐字时间轴，暂停或回跳后高亮仍紧跟播放位置
- 音量平衡从单一开关扩展为关闭、安静、标准、高音量四档，旧设置的开启对应「标准」挡位
- 修复主窗口关闭到托盘后，再次点击应用快捷方式会新开一个应用的问题
- 音量控件上鼠标滚轮改为每次滚动 5%，触控板滑动保持连续变化

### 下载指南

| 处理器架构 | Windows | macOS | Linux |
| :--- | :--- | :--- | :--- |
| amd64 | [安装包 .exe](https://github.com/Amamiya23/mygo-qyin/releases/download/v0.2.2/lilt-setup-0.2.2-amd64.exe) | [安装包 .dmg](https://github.com/Amamiya23/mygo-qyin/releases/download/v0.2.2/lilt-0.2.2-amd64.dmg) | [.deb](https://github.com/Amamiya23/mygo-qyin/releases/download/v0.2.2/lilt_0.2.2_amd64.deb) · [.tar.gz](https://github.com/Amamiya23/mygo-qyin/releases/download/v0.2.2/lilt-0.2.2-linux-amd64.tar.gz) |
| arm64 | [安装包 .exe](https://github.com/Amamiya23/mygo-qyin/releases/download/v0.2.2/lilt-setup-0.2.2-arm64.exe) | [安装包 .dmg](https://github.com/Amamiya23/mygo-qyin/releases/download/v0.2.2/lilt-0.2.2-arm64.dmg) | [.deb](https://github.com/Amamiya23/mygo-qyin/releases/download/v0.2.2/lilt_0.2.2_arm64.deb) · [.tar.gz](https://github.com/Amamiya23/mygo-qyin/releases/download/v0.2.2/lilt-0.2.2-linux-arm64.tar.gz) |

> Linux：Ubuntu / Debian 等支持 deb 的发行版选 `.deb`，其他发行版选 `.tar.gz`。

## 0.2.1

- 以补丁形式修复MyGo Windows 层在多个缩放比例不同的显示器之间恢复窗口时，窗口位置和尺寸可能逐次漂移的问题
- 优化歌单与专辑完整简介浮层的滚动条间距，避免内容与滚动条过于贴近
- 升级上游 MyGo 0.7.4，改善窗口首次显示时的白屏闪烁、原生菜单交互及 macOS 闲置功耗

### 下载指南

| 处理器架构 | Windows | macOS | Linux |
| :--- | :--- | :--- | :--- |
| amd64 | [安装包 .exe](https://github.com/Amamiya23/mygo-qyin/releases/download/v0.2.1/lilt-setup-0.2.1-amd64.exe) | [安装包 .dmg](https://github.com/Amamiya23/mygo-qyin/releases/download/v0.2.1/lilt-0.2.1-amd64.dmg) | [.deb](https://github.com/Amamiya23/mygo-qyin/releases/download/v0.2.1/lilt_0.2.1_amd64.deb) · [.tar.gz](https://github.com/Amamiya23/mygo-qyin/releases/download/v0.2.1/lilt-0.2.1-linux-amd64.tar.gz) |
| arm64 | [安装包 .exe](https://github.com/Amamiya23/mygo-qyin/releases/download/v0.2.1/lilt-setup-0.2.1-arm64.exe) | [安装包 .dmg](https://github.com/Amamiya23/mygo-qyin/releases/download/v0.2.1/lilt-0.2.1-arm64.dmg) | [.deb](https://github.com/Amamiya23/mygo-qyin/releases/download/v0.2.1/lilt_0.2.1_arm64.deb) · [.tar.gz](https://github.com/Amamiya23/mygo-qyin/releases/download/v0.2.1/lilt-0.2.1-linux-arm64.tar.gz) |

> Linux：Ubuntu / Debian 等支持 deb 的发行版选 `.deb`，其他发行版选 `.tar.gz`。

## 0.2.0

- 首页新增更多样式歌曲推荐，对齐官方，并优化首页布局
- Windows 和 Linux 的设置页改为在主窗口中打开，macOS依然对齐系统设计
- 优化底部播放栏的悬停、按压和键盘聚焦反馈，音量面板展开时保持按钮高亮
- 合并上游，优化封面缓存，减少返回已浏览页面时的重复加载，复用播放页仍持有的封面

### 下载指南

| 处理器架构 | Windows | macOS | Linux |
| :--- | :--- | :--- | :--- |
| amd64 | [安装包 .exe](https://github.com/Amamiya23/mygo-qyin/releases/download/v0.2.0/lilt-setup-0.2.0-amd64.exe) | [安装包 .dmg](https://github.com/Amamiya23/mygo-qyin/releases/download/v0.2.0/lilt-0.2.0-amd64.dmg) | [.deb](https://github.com/Amamiya23/mygo-qyin/releases/download/v0.2.0/lilt_0.2.0_amd64.deb) · [.tar.gz](https://github.com/Amamiya23/mygo-qyin/releases/download/v0.2.0/lilt-0.2.0-linux-amd64.tar.gz) |
| arm64 | [安装包 .exe](https://github.com/Amamiya23/mygo-qyin/releases/download/v0.2.0/lilt-setup-0.2.0-arm64.exe) | [安装包 .dmg](https://github.com/Amamiya23/mygo-qyin/releases/download/v0.2.0/lilt-0.2.0-arm64.dmg) | [.deb](https://github.com/Amamiya23/mygo-qyin/releases/download/v0.2.0/lilt_0.2.0_arm64.deb) · [.tar.gz](https://github.com/Amamiya23/mygo-qyin/releases/download/v0.2.0/lilt-0.2.0-linux-arm64.tar.gz) |

> Linux：Ubuntu / Debian 等支持 deb 的发行版选 `.deb`，其他发行版选 `.tar.gz`。

## 0.1.8

- 打磨多处细节
- 新增音量平衡，设置中默认开启，减小不同歌曲之间的音量差异
- 进度条新增「波浪」特效，可在设置中切换标准、烟雾和波浪三种样式
- 支持更改歌曲缓存目录，重启后自动迁移已有缓存
- 改善歌曲、歌手和专辑的搜索稳定性，接口受限时自动切换备用来源，并优化分页和失败重试
- 优化搜索联想，快速输入或切换关键词时及时取消旧请求，避免显示过期建议；搜索历史弹窗与搜索框对齐
- 当前歌曲滚出列表后，可通过定位按钮快速返回；在歌单中再次点击当前歌曲可暂停或继续播放
- 歌单和专辑简介过长时，可通过悬停或键盘聚焦查看完整内容
- 优化播放页、律动和选歌视图中的按钮反馈，悬停时图标更清晰

## 0.1.7

- 修复「今日为你推荐」重复精选歌单的问题，改为独立读取账号首页推荐
- 修正排行榜内容类型筛选，排除无法按歌曲加载的专用榜单，避免打开后显示空列表
- 从歌单返回「为你推荐」或排行榜时恢复原来的滚动位置，前进和重新访问页面也会保留各自的位置
- macOS 刘海播放器新增同步歌词：虚拟刘海收起时显示当前歌词，展开播放时在歌名下显示歌词
- 修复刘海播放器在暂停后拖动播放进度时，歌词未及时刷新的问题

## 0.1.6

- 启用应用内自动更新，支持检查新版本、查看更新说明、下载安装和重启生效；旧版用户需手动安装一次本版本
- 新增微信扫码登录，可在登录窗口切换 QQ / 微信
- 我的主页显示会员状态，支持查询失败后重试
- 「我喜欢」支持单曲 / 专辑切换，专辑页可收藏与取消收藏
- 「我喜欢」单曲与专辑共用大封面页头，切换时保持布局稳定，分别记住滚动位置，并增加标签切换动效
- 侧栏新增「已购音乐」，登录后可分别浏览已购单曲与数字专辑，支持分页、刷新和专辑跳转
- 修复收藏歌单名称显示及取消收藏功能
- 修复登录后歌手页的专辑列表加载失败
- 修复部分歌曲的翻译歌词缺失，以及歌曲评论无法加载和翻页的问题
- 正在播放的标签保留在右侧内容区上方，在切换页面和评论数量变化时保持位置稳定，大窗口中的封面与内容居中显示
- 正在播放的封面、歌名和进度条左侧对齐，调整标签间距并缩短选歌封面墙顶部的模糊过渡
- 修复 Windows 右键菜单宽度计算不足，导致歌手名、专辑名等文字右侧被裁切的问题
- Windows 侧边栏默认使用亚克力，保留已保存的 Mica 选择；macOS 和 Linux 不显示 Windows 材质设置
- 关闭主窗口的设置按平台显示后台运行或托盘选项，Windows/Linux 无法创建托盘时关闭窗口直接退出

## 0.1.5

- 修复登录后搜索歌曲、歌手和专辑返回空结果的问题
- 搜索接口拒绝或返回异常格式时正确报告失败，避免误显示为没有结果
- 项目更名为 Lilt，统一应用名称、安装包、便携包和网站品牌
- 首次启动自动迁移旧 Q 音版的设置、播放队列和缓存，沿用原应用标识以保留登录态
- Windows 侧边栏支持在 Mica / 亚克力间切换，亚克力使用 40% 磨砂遮罩与低对比控件
- 已喜欢歌曲的爱心及点击动效统一使用 Radix 红色，保留绿色主题和未收藏时的空心样式

## QQ 音乐 0.1.4

- 修复歌单歌曲行、侧栏迷你播放器和正在播放详情等位置的封面缺失
- 封面请求使用 QQ CDN 支持的尺寸档位，避免任意尺寸地址返回 404

## QQ 音乐 0.1.3

- 修复 Windows 已配置系统代理时，客户端仍绕过代理直连而导致二维码获取中断的问题
- 登录、曲库、封面与音频下载共用系统代理选择，并遵循环境变量优先级和代理绕过规则
- 取码失败保留具体原因，支持失败后再次刷新并显示新二维码

## QQ 音乐 0.1.2

- 修复登录响应的有效期字段以数字字符串返回时被错误拒绝的问题
- 区分 QQ 音乐服务拒绝码、响应解析、缺失凭据及网络连接错误，便于定位实际失败阶段

## QQ 音乐 0.1.1

- 修复扫码确认失败后错误地址撑出对话框的问题，按登录步骤显示简短提示
- 修复已经扫描时出现错误却无法刷新二维码的问题
- 授权 GET 在 TLS 建连中断、请求尚未发出时重新连接一次
