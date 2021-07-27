package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"path"
	"strings"

	"github.com/go-zookeeper/zk"
)

type ZkSync struct {
	ZkConn *zk.Conn
}

func (s *ZkSync) NeedSyncFile(f string) bool {
	switch path.Ext(f) {
	case ".json", ".yml", ".yaml":
		return true
	}

	return false
}

func (s *ZkSync) CreateParentNodeWhenNoExists(p string) error {
	parentPath := GetParentPath(p)
	if parentPath == "/" {
		return nil
	}

	// 判断是否已经存在
	exists, _, err := s.ZkConn.Exists(parentPath)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	// 尝试创建父节点的父节点
	if err := s.CreateParentNodeWhenNoExists(parentPath); err != nil {
		return err
	}

	// 创建父节点
	if _, err := s.ZkConn.Create(parentPath, nil, 0, zk.WorldACL(zk.PermAll)); err != nil {
		return err
	}

	return nil
}

func (s *ZkSync) SyncNodeWithFile(file string) error {
	// 读取文件内容
	bs, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	// 判断节点是否已经存在
	nodePath := fmt.Sprintf("/%s", file)
	exists, nodeStat, err := s.ZkConn.Exists(nodePath)
	if err != nil {
		return err
	}

	var version int32
	if !exists {
		// 不存在则创建
		if err := s.CreateParentNodeWhenNoExists(nodePath); err != nil {
			return err
		}

		if _, err := s.ZkConn.Create(nodePath, bs, 0, zk.WorldACL(zk.PermAll)); err != nil {
			return err
		}

	} else if nodeStat != nil {
		// 已经存在则更新
		version = nodeStat.Version
		log.Printf("sync node with file version=%d\b", nodeStat.Version)
		if _, err = s.ZkConn.Set(nodePath, bs, version); err != nil {
			return err
		}

	}
	return nil
}

func (s *ZkSync) RemoveNode(file string) error {
	if !strings.HasPrefix(file, "/") {
		file = "/" + file
	}

	exists, stat, err := s.ZkConn.Exists(file)
	if err != nil {
		return err
	}

	if !exists {
		log.Printf("node not exists: file=%s\n", file)
		return nil
	}

	children, stat, err := s.ZkConn.Children(file)
	if err != nil {
		return err
	}
	log.Printf("children of node: node=%s children=%v\n", file, children)

	for _, child := range children {
		child = path.Join(file, child)
		if err := s.RemoveNode(child); err != nil {
			return fmt.Errorf("remove child node error: child=%s error=%s", child, err.Error())
		}
	}

	return s.ZkConn.Delete(file, stat.Version)
}
