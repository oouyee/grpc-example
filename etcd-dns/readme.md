graph TB
    subgraph etcd集群
        A[etcd Server]
    end
    
    subgraph 服务端进程
        B[gRPC Server] -->|Register| C[etcd/register]
        C -->|Put + Lease| A
        D[KeepAlive协程] -->|定期续期| A
    end
    
    subgraph 客户端进程
        E[gRPC Client] -->|Build| F[etcd/resolver]
        F -->|Get| A
        F -->|Watch| A
        A -->|推送变化| F
        F -->|UpdateState| E
    end
    
    B -.服务下线.-> G[Unregister/Delete]
    G --> A
