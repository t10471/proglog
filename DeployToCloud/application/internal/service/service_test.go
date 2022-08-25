package service_test

// func TestService(t *testing.T) {
// 	var services []*service.Service
//
// 	serverTLSConfig, err := tls.SetupTLS(tls.Args{
// 		CertFile:      tls.ServerCertFile,
// 		KeyFile:       tls.ServerKeyFile,
// 		CAFile:        tls.CAFile,
// 		Server:        true,
// 		ServerAddress: "127.0.0.1",
// 	})
// 	require.NoError(t, err)
//
// 	peerTLSConfig, err := tls.SetupTLS(tls.Args{
// 		CertFile:      tls.RootClientCertFile,
// 		KeyFile:       tls.RootClientKeyFile,
// 		CAFile:        tls.CAFile,
// 		Server:        false,
// 		ServerAddress: "127.0.0.1",
// 	})
// 	require.NoError(t, err)
//
// 	for i := 0; i < 3; i++ {
// 		ports := dynaport.Get(2)
// 		bindAddr := fmt.Sprintf("%s:%d", "127.0.0.1", ports[0])
// 		rpcPort := ports[1]
//
// 		dataDir, err := os.MkdirTemp("", "agent-test-log")
// 		require.NoError(t, err)
//
// 		defer func(dir string) {
// 			_ = os.RemoveAll(dir)
// 		}(dataDir)
//
// 		var startJoinAddrs []string
// 		if i != 0 {
// 			startJoinAddrs = append(startJoinAddrs, services[0].Config.BindAddr)
// 		}
//
// 		agent, err := service.NewService(service.Config{
// 			NodeName:        fmt.Sprintf("%d", i),
// 			Bootstrap:       i == 0,
// 			StartJoinAddrs:  startJoinAddrs,
// 			BindAddr:        bindAddr,
// 			RPCPort:         rpcPort,
// 			DataDir:         dataDir,
// 			ACLModelFile:    tls.ACLModelFile,
// 			ACLPolicyFile:   tls.ACLPolicyFile,
// 			ServerTLSConfig: serverTLSConfig,
// 			PeerTLSConfig:   peerTLSConfig,
// 		})
// 		require.NoError(t, err)
//
// 		services = append(services, agent)
// 	}
// 	defer func() {
// 		for _, agent := range services {
// 			_ = agent.Shutdown()
// 		}
// 	}()
//
// 	// wait until services have joined the cluster
// 	time.Sleep(3 * time.Second)
//
// 	leaderClient := client(t, services[0], peerTLSConfig)
// 	produceResponse, err := leaderClient.Produce(context.Background(),
// 		&pb.ProduceRequest{Record: &pb.Record{Value: []byte("foo")}})
// 	require.NoError(t, err)
//
// 	// wait until replication has finished
// 	time.Sleep(3 * time.Second)
//
// 	consumeResponse, err := leaderClient.Consume(context.Background(),
// 		&pb.ConsumeRequest{Offset: produceResponse.Offset})
// 	require.NoError(t, err)
// 	require.Equal(t, consumeResponse.Record.Value, []byte("foo"))
//
// 	followerClient := client(t, services[1], peerTLSConfig)
// 	consumeResponse, err = followerClient.Consume(context.Background(),
// 		&pb.ConsumeRequest{Offset: produceResponse.Offset})
// 	require.NoError(t, err)
// 	require.Equal(t, consumeResponse.Record.Value, []byte("foo"))
// }
//
// func client(
// 	t *testing.T,
// 	agent *service.Service,
// 	tlsConfig *tls.Config,
// ) pb.LogClient {
// 	tlsCreds := credentials.NewTLS(tlsConfig)
// 	opts := []grpc.DialOption{
// 		grpc.WithTransportCredentials(tlsCreds),
// 	}
// 	rpcAddr, err := agent.Config.RPCAddr()
// 	require.NoError(t, err)
// 	conn, err := grpc.Dial(fmt.Sprintf("%s:///%s", loadbalance.Name, rpcAddr), opts...)
// 	require.NoError(t, err)
// 	client := pb.NewLogClient(conn)
// 	return client
// }
