# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

这是一个用于自动更新火山引擎(Volcengine)安全组 SSH 访问规则的工具。当本机公网 IP 发生变化时,自动更新安全组规则,允许新 IP 通过 SSH(22端口)访问。

项目包含两种运行模式:
- **Web 界面模式**(推荐): 提供 Flask Web UI 用于配置和监控
- **独立脚本模式**: 直接运行 `update_ssh_ip.py`

## 核心架构

### 主要组件

1. **`update_ssh_ip.py`**: 核心业务逻辑
   - `SecurityGroupUpdater` 类: 封装所有火山引擎 API 交互
   - 公网 IP 检测: 通过多个公共服务获取(ipify, ifconfig.me 等)
   - 安全组规则管理: 查询、删除旧规则、添加新规则
   - 可独立运行或被 Flask 应用调用

2. **`app.py`**: Flask Web 应用
   - 使用 Flask-APScheduler 实现定时任务
   - 通过 `scheduled_update_task()` 调用 `SecurityGroupUpdater`
   - 自定义 `DBHandler` 将日志写入数据库

3. **`models.py`**: 数据模型(SQLite)
   - `Settings`: 存储配置(AK/SK、区域、安全组ID、检查间隔)
   - `UpdateLog`: 存储运行日志

4. **`templates/`**: Jinja2 模板
   - `base.html`: 基础布局
   - `index.html`: 仪表板(显示状态和最近日志)
   - `settings.html`: 配置页面
   - `logs.html`: 完整日志列表

### 工作流程

```
1. 定时任务触发 (APScheduler, 默认 15 分钟)
   ↓
2. 从多个服务获取当前公网 IP (ipify.org, ifconfig.me, etc.)
   ↓
3. 调用火山引擎 API 查询安全组中 SSH 端口的现有规则
   ↓
4. 比较 IP:
   - 如果相同 → 跳过
   - 如果不同 → 删除旧规则 + 添加新规则 (CIDR: IP/32)
   ↓
5. 记录日志到数据库和文件
```

## 常用命令

### 开发环境设置

```bash
# 使用自动脚本(推荐)
./run.sh

# 手动设置
python3 -m venv venv
source venv/bin/activate  # Windows: venv\Scripts\activate
pip install -r requirements.txt
```

### 运行应用

```bash
# Web 界面模式
python app.py
# 访问 http://localhost:5000

# 独立脚本模式(需先在 main() 中配置 AK/SK)
python update_ssh_ip.py
```

### 生产部署

```bash
# 使用 systemd 服务(Web 模式)
sudo cp volcengine-web.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable volcengine-web
sudo systemctl start volcengine-web
sudo systemctl status volcengine-web

# 查看日志
sudo journalctl -u volcengine-web -f
```

### 数据库操作

```bash
# 数据库文件位置: config.db (SQLite)
# 在 Python 交互式环境中查看/修改数据库
python
>>> from app import app, db, Settings, UpdateLog
>>> with app.app_context():
...     settings = Settings.get_settings()
...     print(settings.region, settings.ssh_port)
```

## 火山引擎 API 集成要点

### API 调用模式

所有火山引擎 VPC API 调用都通过 `volcenginesdkvpc.VPCApi` 实例进行:

```python
# 初始化(在 SecurityGroupUpdater.__init__)
configuration = volcenginesdkcore.Configuration()
configuration.ak = ak
configuration.sk = sk
configuration.region = region
self.api_client = volcenginesdkcore.ApiClient(configuration)
self.vpc_api = volcenginesdkvpc.VPCApi(self.api_client)

# 查询安全组属性
request = volcenginesdkvpc.DescribeSecurityGroupAttributesRequest(
    security_group_id=self.security_group_id
)
response = self.vpc_api.describe_security_group_attributes(request)

# 撤销入站规则
revoke_request = volcenginesdkvpc.RevokeSecurityGroupIngressRequest(
    security_group_id=...,
    protocol='TCP',
    port_start='22',
    port_end='22',
    cidr_ip='1.2.3.4/32',
    policy='accept'
)
self.vpc_api.revoke_security_group_ingress(revoke_request)

# 授权入站规则
authorize_request = volcenginesdkvpc.AuthorizeSecurityGroupIngressRequest(
    security_group_id=...,
    protocol='TCP',
    port_start='22',
    port_end='22',
    cidr_ip='5.6.7.8/32',
    policy='accept',
    priority='1',
    description='SSH access - Auto updated by script'
)
self.vpc_api.authorize_security_group_ingress(authorize_request)
```

### 关键注意事项

1. **规则匹配逻辑**: 在 `get_ssh_rule_from_security_group()` 中,必须同时匹配:
   - `direction == 'ingress'`
   - `port_start <= SSH_PORT <= port_end`
   - `protocol.lower() == 'tcp'`

2. **CIDR 格式**: 始终使用 `/32` 表示单个 IP 地址

3. **删除+添加模式**: 火山引擎要求先撤销旧规则,再授权新规则(不支持直接修改)

4. **异常处理**: 所有 API 调用都应捕获 `volcenginesdkcore.rest.ApiException`

## Flask 调度器集成

APScheduler 在 Flask 中的集成模式:

```python
# 初始化
scheduler = APScheduler()
scheduler.init_app(app)
scheduler.start()

# 添加定时任务
scheduler.add_job(
    id='ip_update_job',
    func=scheduled_update_task,
    trigger='interval',
    seconds=settings.check_interval  # 默认 900 秒
)

# 重新配置任务(更新间隔时)
scheduler.remove_job('ip_update_job')
scheduler.add_job(id='ip_update_job', func=..., trigger='interval', seconds=new_interval)

# 手动触发一次任务
scheduler.add_job(id=f'manual_run_{datetime.now().timestamp()}', func=scheduled_update_task)
```

**注意**: `scheduled_update_task` 必须在 `app.app_context()` 中执行数据库操作。

## 日志系统

双重日志机制:

1. **文件日志**: `update_ssh_ip.log` (由 `update_ssh_ip.py` 的 `logging.basicConfig` 配置)
2. **数据库日志**: `UpdateLog` 表(通过自定义 `DBHandler` 写入)

在 `app.py` 中,将 `DBHandler` 添加到 `update_ssh_ip` 模块的 logger:

```python
updater_logger = logging.getLogger("update_ssh_ip")
updater_logger.addHandler(DBHandler())
```

这样 `update_ssh_ip.py` 中的所有 `logger.info()` 调用都会同时写入文件和数据库。

## 安全配置

- **敏感信息**: AK/SK 存储在 `config.db` 中,确保不提交到版本控制
- **SECRET_KEY**: `app.config['SECRET_KEY']` 在生产环境中应使用环境变量或安全的随机值
- **Web 服务器**: 生产环境建议使用 Gunicorn + Nginx,而非 Flask 开发服务器
- **权限最小化**: 建议为脚本创建火山引擎子账号,仅授予 VPC 安全组读写权限

## 依赖关系

核心依赖及其用途:
- `volcengine-python-sdk`: 火山引擎官方 SDK
- `requests`: HTTP 请求(获取公网 IP)
- `Flask`: Web 框架
- `Flask-SQLAlchemy`: ORM 数据库操作
- `APScheduler`: 定时任务调度

## 扩展建议

添加新功能时的建议:
- 新增 IP 查询服务: 在 `IP_SERVICES` 列表中添加
- 支持多端口: 修改 `get_ssh_rule_from_security_group()` 的过滤逻辑
- 添加通知: 在 `check_and_update()` 中集成邮件/Webhook 通知
- 支持多安全组: 将 `security_group_id` 改为列表,循环处理
