# Morph morph-v2.2.8 合并验证报告

日期：2026-11-11

结论：本轮合并与配套服务验证通过，PR保持open等待review/merge。

PR：https://github.com/Chaintable/go-ethereum-morph-x/pull/25

## 1. Release 更新与升级原因

官方没有声明本次为强制升级或设置截止日期。geth morph-v2.2.8 自身仅删除旧 Morph block-trace RPC 别名，没有同步、执行或共识修复。node v0.6.5 隐含的 Tendermint v0.3.11 更新，为 blocksync 与 sequencer 广播/同步通道增加完整解码前的重复元素数量检查；这是安排本次配套验证的主要可靠性收益，不将其描述为已证实现场攻击或已公告 CVE。

官方发布内容全部如下：geth 删除 morph namespace 下的 block trace API；node 移除 tx-submitter 中未使用的 GetBlockTraceByNumber 接口，并将 devnet 所有 fee token 注册为 call mode。node/go.mod 与实际官方镜像二进制均确认 geth 依赖1c528b35（morph-v2.2.8）、Tendermint v0.3.11；devnet 修改不改变主网已部署的 token 注册。

本次删除的是 `morph` namespace 下的全部 tracer 别名。旧注册将 `*API` 转成 `TraceBlock` 接口，但 RPC 注册按动态类型 `*API` 反射，所以实际删除的普通 RPC 共13个：`morph_debankBlock`、`morph_getBlockTraceByNumberOrHash`、`morph_getTxBlockTraceOnTopOfBlock`、`morph_intermediateRoots`、`morph_standardTraceBadBlockToFile`、`morph_standardTraceBlockToFile`、`morph_traceBadBlock`、`morph_traceBlock`、`morph_traceBlockByHash`、`morph_traceBlockByNumber`、`morph_traceBlockFromFile`、`morph_traceCall`、`morph_traceTransaction`；另移除经 `morph_subscribe("traceChain", ...)` 访问的订阅。等价方法仍在 `debug` namespace，独立的 `trace_debankBlock` 保留。ethclient helpers 和 console bindings 同时移除。生产是否存在使用这些旧别名的外部消费者尚未验证，上线前需核对调用方/网关访问日志；不能将“源码中无消费者”当作“实际没有调用”。

来源：[geth release](https://github.com/morph-l2/go-ethereum/releases/tag/morph-v2.2.8)、[node release](https://github.com/morph-l2/morph/releases/tag/v0.6.5)、[Tendermint diff](https://github.com/morph-l2/tendermint/compare/v0.3.10...v0.3.11)。官方 run-morph-node 的 main 部署模板本轮仍 pin geth2.2.6/node0.6.3，滞后于 Celadon；配套版本以 node0.6.5 的依赖 pin 为准。

## 2. Merge 冲突与影响

生产 writer/seed 均运行 v2.2.7-ct.1 + node0.6.4；生产 geth tag 已被本次 main 基线 ad2de3152 包含。合并 upstream1c528b35 后 commit3c3627eda，只改5文件、+1/-48。唯一冲突 eth/tracers/api.go 保留共享 API 实例及独立 trace namespace 下的 DebankAPI 注册。

api.go 除移除旧 morph 注册外与 main 逐字一致，其余4个变更文件与目标 upstream tag 逐字一致；其他 fork 文件、pipeline v0.0.70-morph-v2.2.4-ct.2、EOA账户/state diff失败传播、retention 和既有回归测试均未改变。上游未改 StateDB、执行器或 tracer hooks，不引入新的 fork 同步执行路径。git diff --check 通过。

## 3. 测试部署

本次隔离 EC2 测试部署仅运行 geth PR 镜像428e2b18与官方 node0.6.5，不运行 ETL/Kafka/S3 客户端；不注入 AWS 凭证，禁用 IMDS 凭证回退。本轮验证 RPC 产物，不称为下游上传验证。

源为2026-11-11 02:02 UTC 的 seed 快照，独立350GiB gp3卷、6000IOPS/250MiB/s、初始化300MiB/s。ext4卷根按生产方式挂到/var/data，使用/var/data/geth-data、/var/data/node-data与共用jwt；首次启动前清除geth nodekey、node_key.json及priv_validator_key.json。初始实占114GiB、可用214GiB。

geth8GiB/4CPU、node4GiB/2CPU；独立10.28.18.0/24；RPC/metrics仅绑定127.0.0.1。PR synthetic merge428e2b18与被测head的tree一致；CI42979686449的amd64、arm64、manifest均成功，公共registry已确认镜像存在。

## 4. 验证结果

| 检查 | 结果 |
|---|---|
| go build ./...（Go1.27.1/macOS arm64） | PASS |
| 四项DeBank tracer/state diff回归 | PASS：TestDebankBlockIncludesEOAStateDiff、TestDebankBlockStateDiffMatchesCanonicalState、TestAddERC20BalanceDynamicSlotDiscovery、TestSyntheticPrecreditMaskPreservesPrestateTracerView |
| params、rollup/fees包测试 | PASS |
| ethclient包测试 | 两项基线失败，见下节 |
| node derivation/types/version包测试 | PASS |
| P2P交易数量边界 | 0/100000接受、100001拒绝 |
| PR amd64/arm64/manifest CI | PASS |
| 官方RPC20块hash | 20/20 MATCH，29617736–29617755 |
| 固定100块hash/stateRoot/receiptsRoot | 100/100 MATCH |
| 完整trace_debankBlock与state diff | 100/100 MATCH |
| 两个旧GetBlockTrace别名移除、对应debug方法注册 | PASS；debug参数错误探测仅证明注册，不证明成功执行；自定义trace由100块实际调用覆盖 |
| 优雅停止/重启 | 两进程exit0、OOM=false；约3秒完成停止和恢复 |
| 追平与batch校验 | 07:42:31Z测试/生产latest29629937、safe29629676、finalized29628918及hash全部一致 |

固定100个不同样本范围29277502–29629943：Celadon第一块29277512前10块及从该块起10块、快照高度29617736起20块、追块段高gas40块和近head20块。其中52个非空块、100笔交易，最高gasUsed=1,786,167。52块非空state diff逐一解码全部六字段并按集合比较；48块返回协议规定的0x，额外确认其parent/current state root相同。完整产物仅忽略process_start_timestamp，并按既有约定规范化map生成的集合顺序。原始响应完整保留。

geth首次启动07:34:50Z，node启动07:36:02Z；07:36:45–48Z优雅停止重启，随后继续导入。07:42:31Z确认两层追平，从快照累计新增执行12,201块。geth启动至完整进度核对约7分40秒，包含初始化与一次重启。本轮100块采集比较于07:47:11Z完成，耗时134.59秒；前期准备、首轮执行器修正和其他并行验证不计入该RPC轮次耗时，不累加成端到端墙钟时间。

节点从Celadon激活后的11月11日快照启动，RPC回放覆盖11月4日分叉边界前后；这证明选定历史块的执行输出一致，不等于服务同步跨越实际分叉激活过程。测试没有运行ETL或生产上传/通知链路。

## 5. 其他问题与覆盖边界

Claude独立只读审查未发现P0/P1；已复核并补全P2 API删除范围说明，以及P3 debug注册探测的覆盖表述。尚未检查外部旧别名流量。用户在本轮完成后将后续验证标准提高到1000块；上述记录仍是本轮实际100块，不改写历史数量。

ethclient整包的TestToFilterArg与TestEthClient失败已在合并前ad2de3152基线复现，未跳过或修改测试；不能称全包测试通过。其他历史失败保留于此前2.2.7报告，本轮仅声明下表实际执行的覆盖。

追赶时node出现finalized > safe日志64次；node/derivation/tag_advance.go与v0.6.4逐字一致，flushTags在L1 finalized推进而历史safe校验尚未追齐时直接return，不调用SetBlockTags。追平后safe/finalized与生产一致。另保留部分peer拨号超时、短暂历史HeaderByNumber not found及geth启动时pre模块未注册提示，不能称所有日志零错误。捕获窗口未见panic/FATAL或state/batch/blob hash不一致。

首次trace执行器错误要求每块diff非空，在首个空diff上停止；核对固定pipeline的GetOutPut约定后，保留首次证据并使用同一冻结100块完成复跑，不换样本。该中断不作为产品不一致，也没有放宽非空diff的六字段检查。

P2P数量检查采用固定依赖包的直接边界调用，证明BlockResponseV2交易数量0/100000被接受、100001被拒绝；不等同于对恶意peer的完整网络压力或DoS验收。真实服务同步用于检查正常流量兼容性。未执行生产变更、PR合并、release发布、metadata回写或清理。

### 本轮实际 Compose

```yaml
services:
  morph-geth:
    image: public.ecr.aws/b2h7a5c4/chaintable/morph-writer:428e2b18
    container_name: morph-v228-morph-geth
    restart: unless-stopped
    stop_grace_period: 2m
    logging:
      driver: json-file
      options:
        max-size: 100m
        max-file: '5'
    mem_limit: 8g
    cpus: 4
    user: 0:0
    entrypoint:
    - /app/geth
    command:
    - --morph
    - --datadir=/var/data/geth-data
    - --http
    - --http.addr=0.0.0.0
    - --http.port=8545
    - --http.api=net,eth,engine,debug,pre,trace,morph
    - --http.vhosts=*
    - --http.corsdomain=*
    - --ws
    - --ws.addr=0.0.0.0
    - --ws.port=8546
    - --ws.api=net,eth,engine,debug,pre,trace,morph
    - --syncmode=full
    - --gcmode=archive
    - --snapshot=false
    - --cache=4096
    - --authrpc.addr=0.0.0.0
    - --authrpc.port=8551
    - --authrpc.jwtsecret=/var/data/jwt.txt
    - --authrpc.vhosts=*
    - --txlookuplimit=0
    - --override.morph203time=1747029600
    - --metrics
    - --pprof
    - --pprof.addr=0.0.0.0
    - --pprof.port=9260
    volumes:
    - /opt/apps/morph-merge-v228/writer:/var/data
    ports:
    - 127.0.0.1:28545:8545
    - 127.0.0.1:28546:8546
    - 127.0.0.1:28551:8551
    - 127.0.0.1:29260:9260
    networks:
    - morph-v228-net
    environment:
      AWS_EC2_METADATA_DISABLED: 'true'
  morph-node:
    image: ghcr.io/morph-l2/node:0.6.5
    container_name: morph-v228-morph-node
    restart: unless-stopped
    stop_grace_period: 1m
    logging:
      driver: json-file
      options:
        max-size: 100m
        max-file: '5'
    mem_limit: 4g
    cpus: 2
    user: 0:0
    command:
    - --mainnet
    - --home=/var/data/node-data
    - --l1.rpc=http://trace.eth.blockchain
    - --l1.beaconrpc=http://archive-beacon.eth.blockchain
    - --l2.engine=http://morph-geth:8551
    - --l2.eth=http://morph-geth:8545
    - --l2.jwt-secret=/var/data/jwt.txt
    volumes:
    - /opt/apps/morph-merge-v228/writer:/var/data
    networks:
    - morph-v228-net
    depends_on:
    - morph-geth
    environment:
      AWS_EC2_METADATA_DISABLED: 'true'
    entrypoint:
    - morphnode
networks:
  morph-v228-net:
    driver: bridge
    ipam:
      driver: default
      config:
      - subnet: 10.28.18.0/24
```
