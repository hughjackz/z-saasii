#csms-SAAS

基本拓扑为前端(vue)+后端(go+cpp)+mysql数据库

其中后端分为web后端和ocpp后端

1. 需要实现基本功能为
1.1 前端负责用户对ocpp连接设备的基本操作、用户管理。
1.2 web服务器后端负责处理前端指令，用户数据库、ocpp设备数据库的维护，ocpp后端服务器负责与物联网中的ocpp设备进行通讯

2. 前端实现的具体功能为
    UI采用简约风即可，参考gemini style
2.1 用户登录
2.2 用户权限管理
    显示不同的功能模块（由后端根据前端请求动作中的用户名进行判断）
    用户相关信息不能放在URI请求中
2.3 页面布局
    基本分为
    *侧边栏，显示功能模块 
    *顶部栏显示当前功能模块的名字、设备名称、时间、用户名 
    *底部栏显示公司名称 
    *中间为内容显示框图

    2.3.1 OVERVIEW
        显示此用户全部的设备[前端请求，由后端告知]，及各个设备的状态，登录时间等信息
    
    2.3.2 OCPP 以下功能模块也由后端告知
        内容栏中构建顶部栏，如果是CS_Admin，需要有一个CP_OP选择栏，和device选择栏，如果是CP_OP或CP_OM登录时，仅需要一个device选择栏，
        在下述的功能模块中，有的需要向后端的发送请求，此时将CP_OP和device作为参数传送进去，以便后端能够明确知道待操作的设备
        
        2.3.2.1 configuration
            有近100个配置数据的显示，这些key和value来自后端
            支持单个配置、部分配置、所有配置的读取和设置

            第一次登录到管理后台的，因为没有相关缓存数据，要全部获取
            后续操作，支持单个的读取配置和修改数据，也支持部分配置的读取和设置

        2.3.2.2 transaction
            2.3.2.1 bill
                显示过往的订单
                显示当前正在运行的订单
                订单中的数据有
                    设备号 chargepointid
                    连接号 connectorid
                    订单号 transactionid
                    启动时间 starttime
                    结束时间 stoptime
                    充电时长 duration
                    启动电能 startmeter
                    结束电能 stopmeter
                    消费电能 costenergy
            2.3.2.2 action
                remote start，需要填入设备的连接号，启动button
                remote stop， 需要填入设备的连接号和交易id，停止button

        2.3.2.3 maintaince
            包含Hard Rest button，并click之后弹出确定的对话框之后，发送给后端Reset指令
            OTA升级，支持加载本地文件，将文件上传之后端服务器
            日志下载触发button，并click之后弹出确定的对话框之后，发送给后端Getlog指令

        2.3.2.4 PNC 实现15118-2 P&C功能backend/doc/ocpp1.6/protocol/ocpp_1_6_ISO_15118_v10.pdf，对应本章节Part4.2.9描述
            a) 安装SECCLeaf证书
                --操作者需要选择使用具体的V2G root,CPO sub1, CPO sub2证书作为签发者,有触发设备请求安装证书的button

            b) 安装ROOT证书，
                --需要向后端请求所有可用的Mo root、V2G root证书，
                --从证书库中选择安装，操作者先选择证书类型，然后UI根据证书类型，显示对应所有证书，让操作者选择（支持下拉多选），然后将所选的证书list给到后端

            c) 支持获取设备的已安装的全部证书，或者选择证书类型Mo root，V2G root， SECCLeaf，向后端发送请求，然后等到设备回复后，需要显示在此界面

            d) 删除设备安装的证书，UI提供证书类型筛选框，方便用户选择，选好要删除的证书后，向后端发送请求
                --需要向后端请求所有可用的Mo root、V2G root证书，和属于已选设备的SECC证书
                --操作者先选择证书类型，然后UI根据证书类型，显示对应该类型的所有证书，操作者选择具体的证书，然后前端将所选证书发送给后端,后端发送deletecertificates给设备
            
            e) 选择签发车辆合同证书所需的证书组(4.2.9.5要用到)
                --签发合同证书需要2.3.4.2中除了secc-leaf的所有证书类型各一个
                --选择证书类型，然后向后端请求该类型的证书列表，用户在下拉框中确定一个该类型的证书
                --将所有的类型的证书选择确定好之后，提交给后端
        2.3.2.5 Smartcharging

    2.3.3 VDV261
        内容栏中构建顶部栏，如果是CS_Admin，需要有一个CP_OP选择栏，如果是CP_OP或CP_OM登录时，不需要有筛选器
        2.3.3.1 VDVProfile
            a) 读取  用户在前端点击read button，请求后端CP_OP名下所有的VDVprofile
            b) 新建  name、CP_OP(下拉选择)、request中的driveoff、prec_dsrd、prec_hvac、ambienttemp，post到后端，后端存入数据库
            c) 修改  每行数据都有 mod action操作，允许修该profile的以上内容
            d) 删除  每行数据都有 del action操作，允许删除该profile

        2.3.4.2 VDVCarManagement
            CP_OP和CP_OM只有读取的权限，没有新建和修改的权限
            a) 读取  用户在前端点击read button，请求后端CP_OP名下所有的Carinfo
            b) 新建  VIN、password、EVCCID(车辆上传报文时更新，不强制要求)、odo(车辆上传报文时更新不强制要求)，VDVProfile（下拉选择），CP_OP选择
            c) 修改  每行数据都有 mod action操作，允许修该car的以上内容
            d) 删除  每行数据都有 del action操作，允许删除该car信息

        2.3.4.3 VDVSetting (仅有CS_Admin有权限进行操作)，针对于全局的配置，不是单一的CP_OP用户
            a) 支持上传和下载VDVroot证书、VDVserver证书、VDVserver私钥，文件分别为resource/admin/cert/VDVroot.pem、resource/admin/cert/VDVserver.pem、resource/admin/cert/VDVserver.key
            b) 可以选择IPV4、IPV6、IPV4&IPV6dual三种模式
            c) 支持重启后端的VDV261服务
    2.3.4 Mangement
        2.3.4.1 users
            能够实现用户的增删改查，
            基本参数有用户权限、用户名称、联系方式、Email、公司名称、使能等
            用户有三种，
            
            CS_Admin   CS_Admin只需要管理CP_OP用户、用户权限即可，不需要对其数据（CP_OM、device、idtag、certificate）进行管理
            CP_OP   运营商 ：可以查看、修改、新建自己名下的CP_OM数据,和自己名下所有的CP_OM所属的所有CP_OM、device、idtag、certificate数据
            CP_OM   运维人员：CP_OM的功能和CP_OP的功能应该一致，但是没有用户管理的权限，只有修改密码的权限，CP_OM的权限由CP_OP给予，但是不应该超过CP_OP自身的权限。

            其中CS_Admin可以对CP_OP、CP_OM、CP_OP可以对CP_OM的功能权限做管理，可以限制其使用和关闭某些功能，功能模块见2.3.2和2.3.3
            在CS_Admin建立CP_OP用户时，后端应该同时建立resource/{{CP_OP_name}}/certificate/、resource/{{CP_OP_name}}/deviceFirmware/ 、resource/{{CP_OP_name}}/deviceLog/文件夹
 
        2.3.4.2 certificates
            包含以下类型的证书{{cert.type}}=
            >V2G-root-cert
            >CPO-sub1-cert
            >CPO-sub2-cert、CPO-sub2-key


            >CPS-sub1-cert
            >CPS-sub2-cert
            >CPS-leaf-cert、CPS-leaf-key，

            >MO-root-cert
            >MO-sub1-cert
            >MO-sub2-cert
            >Contract-leaf-cert、Contract-leaf-key

            >OEM-root-cert
            >OEM-sub1-cert
            >OEM-sub2-cert

            >SECC-leaf-cert

            前端界面上传证书（证书本地上传、上传类型由用户自己选择）
                a. 选择证书组类型，即{{cert.type}}，(如果是CS_Admin,还需要选择CP_OP)，
                b.（SECC-leaf-cert不作为上传选项）
                c. CPO-sub2、CPS-leaf、Contract-leaf必须同时上传证书和私钥，密码作为可选
                d. 后端不需要判断证书是否合法(因为可能会上传需要调试的错误证书)，只需解析相关参数即可[SerialNumber、issuerName、Validity、SubjectName、SubjectPublicKey、SignatureAlgorithm]，
                    然后将相关证书关参数和CP_OP.name、certType、owncontent、privatekeyContent、enabled、password一起存入数据库
                    CPO-sub2、CPS-leaf、Contract-leaf私钥不需要单独存入数据库，但是需要作为证书的相关参数和password放入数据库中
                e. 本地不存放证书文件和私钥文件，全部交由数据库管理

            前端证书显示
                1.当用户进入此界面时，不需要立刻向后端请求证书显示，应该有Read button触发
                    --需要有证书筛选选项，选项有CP_OP.name(仅当CS_Admin用户登录显示)，证书类型{{cert.type}}
                    --SECCLeaf证书为该CP_OP名下签发过的所有证书，
                2.显示的条目为证书名称、证书类型、证书所属的CP_OP.name, action,
                    --action包括check、delete
                    --action.check可以查看其对应的详细信息
                    --action.content可以查看其完整的内容
                    --action.delete删除对应的证书文件、DB数据

        2.3.4.3 devices
            能够实现设备的增删改查，
            基本参数有devicename、protocol、location、enable、heatbeatinterval，所属的CP_OM(仅由CP_OP、CS_Admin新建时需要选择)等

        2.3.4.4 Idtag
            支持新建、修改、删除idtag，
            基本参数有有parentidtag，status（block、valid、expired），expirytime，

        2.3.4.5 profile
            支持导入智能充电配置文件，并支持重命名
            支持删除某个已有的充电配置文件

    2.3.5 EVENT
        查看系统的日志，操作步骤为
        1.如果是CS_Admin 需要先选择需要那个CP_OP，如果是CP_OP、CP_OM则无此筛选框
        2.用户可以选择查看两种日志--实时日志、过去某天的日志
        3.点击Read button


3. web后端实现的功能
    3.1 对前端的各个功能模块及请求做路由处理
    3.2 对前端的登录行为做权限管理，以及jwt有效期管理
    3.3 将前端具有权限相关的数据、用户私有的数据放置在数据库中
    3.4 将前端的OCPP相关请求和ocpp后端做交互
    3.5 web后端和ocpp后端使用内部socket通讯方式，ocpp后端作为server, 数据格式使用json
        基本格式为
        {
            "CP_OP_name": string,
            "CP_OM_name": string,
            "Action": string,
            "payload": object}

    3.6 后端加入日志系统，按照每天进行记录，按照CP_OP进行分别记录，文件放置在backend/mylog目录下
        1.需要记录前端对后端的交互行为和数据
        2.需要记录后端和device的交互数据
        3.需要记录日常的错误信息
        4.针对前端请求查看日志的数据进行响应




4. ocpp后端功能
    现在要完善ocpp后端的部分，使用gorilla/websocket和device进行通讯
    分为三个websocket协议，分别为ocpp1.6、ocpp2.0.1、ocpp2.1
    目前先实现ocpp1.6部分，设备连接url的路由基本构成为ws://ip:port/csocpp{{version}}/{{CP_OP.name}}/{{device.name}}
    根据csocpp{{version}}/{{CP_OP.name}}/{{device.name}}，找到对应的device.id作为操作句柄

    4.1 db.device相关数据说明
        db.device.enabled 代表平台是否同意此桩接入到后端，在4.3.1中使用 0-rejected 1-accepted
        db.device.status 代表了桩的当前的状态（前提是已经同意注册）

    4.2 OCPP1.6相关功能描述
        相关接口应该和主流程独立package出来，为了后面实现和OCPP2.0.1、OCPP2.1的多版本兼容处理
        4.2.1 bootnotification
        4.2.2 heartbeat
        4.2.3 statusnotification
        4.2.4 triggermessage
        4.2.5 authorize
        4.2.6  getConfiguration & changeConfiguration
            后端然后将接收到的response中的数据存储到内存中，不放置在数据库中
            根据前端的请求的body，上传或者修改全部数据或者部分数据
        4.2.7 transaction
            4.2.7.1 remotestarttransaction
            4.2.7.2 remotestoptransaction
            4.2.7.3 starttransaction
                    需要自行生成订单号（int类型，从0自增即可），然后发送给device
            4.2.7.4 stoptransaction
        4.2.8 metervalue

        4.2.9 15118 PNC
            关于15118 P&C功能，协议文档为backend/doc/ocpp1.6/protocol/ocpp_1_6_ISO_15118_v10.pdf，后端需要分模块实现，不应该放在同一个文件中
            通讯报文的数据结构采用datatransfer，其中根据messageid区分功能模块，payload为具体的数据内容，payload的schema在"backend/doc/ocpp1.6/schema/PNC/"
            4.2.9.1 SECCLeaf证书签发，对应协议文档中Part3
                1> 用户在UI前端请求device触发安装证书流程，此时需要将前端操作者选用的V2G root、V2G sub1、V2G sub2证书记录下来
                    saas----TriggerMessage.req--->device
                    saas<---TriggerMessage.conf----device
                2> 设备请求SAAS安装, payload中包含了CSR文件
                    后端根据刚才记录的V2G sub2证书对CSR文件进行签发，
                    签发时应该会有serialnumber，此参数需要放在db中，下次签发自增+1(type=SECCLEAF)，初始值应为0x13155BC
                    生成的证书文件名称命名为{{device.name}}_SECCLeaf_{{serialNumber}}.pem，放置在resource/{{CP_OP_name}}/certificate/
                    然后将所有的相关数据存放在数据库中
                    saas<---SignCertificate.req----device
                    saas----SignCertificate.conf--->device

                3> SAAS将签发过的证书发送给device
                    saas----CertificateSigned.req--->device
                    saas<---CertificateSigned.conf----device

            4.2.9.2 获取设备已经安装的证书，将前端请求查询的证书类型，通过GetInstalledCertificateIds指令下发给device，
                    得到device的回复后，回复给前端
                    saas----GetInstalledCertificateIds.req--->device
                    saas<---GetInstalledCertificateIds.conf----device

            4.2.9.3 删除证书，将前端请求的证书名字，对应查询到数据库中的证书，得到其hashAlgorithm、issuerNameHash、issuerKeyHash、serialNumber后，下发给device
                    saas----DeleteCertificate.req--->device
                    saas<---DeleteCertificate.conf----device

            4.2.9.4 安装ROOT证书，将前端请求安装的证书list解析，然后找到对应的具体内容，然后逐个下发给device，不能放在一帧报文中.
                    saas----InstallCertificate.req--->device
                    saas<---InstallCertificate.conf----device

            4.2.9.5 ContractCertificate安装，后端接收到请求后，解析在payload中exirequest，预留一个ContractGenerate接口，入参为exirequest、certificate List（2.3.2.4 e中确定的证书组）,出参为exirequest string
                    saas<---Get15118EVCertificate.req----device
                    saas----Get15118EVCertificate.conf--->device
            
            4.2.9.6 authorize ，
                    1.device请求过来的PNC.authorize中的certificate和iso15118CertificateHashData是2选1的，要能够对其进行正确的解析，
                    如果是certificate信息，后端需要对证书证书链信息进行解析出上级签发者，然后根据签发者的信息寻找数据库中是否有对应，并对其验证，certificateStatus根据结果返回具体状态
                    如果是iso15118CertificateHashData, 后端需要根据其issuerNameHash、issuerKeyHash找到对应的签发者是否在数据库中，如果是在数据库中，certificateStatus返回accepted
                    2.解析idtoken数据，查询idtag数据库是否存在，返回idTokenInfo给device
                    saas<---Authorize.req----device
                    saas----Authorize.conf--->device



5. 数据库结构
    database名字:db-saas-ocpp-dev,使用innodb

    每一个CP_OP都应该有一张dataBase，这样的话方便对其用户、devidce进行管理
    有以下table
    5.1 "role" 用户管理
    5.2 "action" 用户的操作权限以及相应API
    5.3 "transaction" 用户的交易数据
    5.4 "device" 设备管理
    5.5 "certificate" 证书管理
    5.6 "idtag"管理
    5.7 "VDVProfile" id[uuid]、name[string]、driveoff[hh:mm]、prec_dsrd[int]、prec_hvac[int]、ambienttemp[int]，{{CP_OP_name}}
    5.8 "VDVCarInfo" id[uuid]、VIN[string]、password[string]、EVCCID[string]、odo[int]，VDVProfile_id，{{CP_OP_name}},租户ID

6.VDV261后端功能
    6.1　实现的功能
        VDV261后端主要是对VDV261协议涉及的EVCC与VDV261后台的交互做处理。并提供VDV261相关的配置项控制。
    6.2 和前端交互
        6.2.1 新增carinfo
            VIN码信息是唯一的，如果新建重复的VIN，需要返回前端错误
        6.2.2 允许前端重启VDV261服务
        6.2.3 允许前端配置网络模式、证书相关文件

    6.3　和车端消息交互
        VDV261的消息分为请求与应答。
        6.3.1 HTTP Basic认证。
            EVCC会将身份信息组合，并填入authorization头部。对于VDV261后台来说，身份信息为：{{VIN}}:{{password}}
            身份信息会组合成一个字符串，形式为vin:password。例如：vin为camaevcc01，password为123456。那么身份信息应为：camaevcc01:123456。
            在拦截到VDV261登录时，做如下判断操作：
            a)	判断authorization是否正确，如不正确，登录失败，响应401；
            b)	判断vin及password是否在数据库表中，如不在，登录失败，响应401；
        
        6.3.2 可选择IPV6和IPV4的https功能
            a) 网络模式有前端配置实现，保存至yaml文件，默认使用IPV6
            b) https的相关证书和私钥使用resource/admin/cert/VDVserver.pem和resource/admin/cert/VDVserver.key，如果没有，不开启服务

        6.3.3　请求信息：EVCC上报请求消息到后端，(连接信息可配置)，使用POST
            URL：https://domain:port/vdv
            内容类型：json 请求参数如下 见backend/doc/VDV261/schema/VDV261req.json
        6.3.4　应答消息：后台下发配置给EVCC。
            内容类型：json 应答参数如下 见backend/doc/VDV261/schema/VDV261resp.json
            后端通过EVCC上报过来的VIN信息，查询数据库中该VIN对应的VDVprofile，然后回复给EVCC
            关于VDV的请求和回复也记录到log中    
 