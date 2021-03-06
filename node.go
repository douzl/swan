package main

import (
	"github.com/Dataman-Cloud/swan/src/agent"
	"github.com/Dataman-Cloud/swan/src/config"
	"github.com/Dataman-Cloud/swan/src/manager"

	"github.com/boltdb/bolt"
	"golang.org/x/net/context"
)

const (
	MODE_MANAGER = "manager"
	MODE_AGENT   = "agent"
	MODE_MIXED   = "mixed"
)

type Node struct {
	agent   *agent.Agent      // hold reference to agent, take function when in agent mode
	manager *manager.Manager  // hold a instance of manager, make logic taking place
	config  config.SwanConfig // swan config
	ctx     context.Context
}

func NewNode(config config.SwanConfig, db *bolt.DB) (*Node, error) {
	m, err := manager.New(config, db)
	if err != nil {
		return nil, err
	}

	a, err := agent.New(config)
	if err != nil {
		return nil, err
	}

	node := &Node{
		config:  config,
		manager: m,
		agent:   a,
	}

	return node, nil
}

func (n *Node) Start(ctx context.Context) error {
	if n.config.Mode == MODE_MANAGER || n.config.Mode == MODE_MIXED {
		if err := n.runManager(ctx); err != nil {
			return err
		}
	}

	if n.config.Mode == MODE_AGENT || n.config.Mode == MODE_MIXED {
		if err := n.runAgent(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (n *Node) runAgent(ctx context.Context) error {
	return n.agent.Start()
}

func (n *Node) runManager(ctx context.Context) error {
	managerCtx, cancel := context.WithCancel(ctx)
	n.manager.CancelFunc = cancel
	return n.manager.Start(managerCtx)
}

func (n *Node) stopManager() {
	n.manager.Stop(n.manager.CancelFunc)
}
