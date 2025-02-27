package zookeeper

import (
	"github.com/go-zookeeper/zk"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"mosn.io/layotto/components/lock"
	"mosn.io/pkg/log"
	"testing"
	"time"
)

const resouseId = "resoure_1"
const lockOwerA = "p1"
const lockOwerB = "p2"
const expireTime = 5

var cfg = lock.Metadata{
	Properties: make(map[string]string),
}

func TestMain(m *testing.M) {

	cfg.Properties["zookeeperHosts"] = "127.0.0.1;127.0.0.1"
	cfg.Properties["zookeeperPassword"] = ""
	m.Run()

}

// A lock ,A unlock
func TestZookeeperLock_ALock_AUnlock(t *testing.T) {

	comp := NewZookeeperLock(log.DefaultLogger)
	comp.Init(cfg)

	//mock
	ctrl := gomock.NewController(t)
	unlockConn := NewMockZKConnection(ctrl)
	lockConn := NewMockZKConnection(ctrl)
	factory := NewMockConnectionFactory(ctrl)
	path := "/" + resouseId
	factory.EXPECT().NewConnection(time.Duration(expireTime)*time.Second, comp.metadata).Return(lockConn, nil).Times(2)

	lockConn.EXPECT().Create(path, []byte(lockOwerA), int32(zk.FlagEphemeral), zk.WorldACL(zk.PermAll)).Return("", nil).Times(1)
	lockConn.EXPECT().Close().Return().Times(1)

	unlockConn.EXPECT().Get(path).Return([]byte(lockOwerA), &zk.Stat{Version: 123}, nil).Times(1)
	unlockConn.EXPECT().Delete(path, int32(123)).Return(nil).Times(1)

	comp.unlockConn = unlockConn
	comp.factory = factory

	tryLock, err := comp.TryLock(&lock.TryLockRequest{
		ResourceId: resouseId,
		LockOwner:  lockOwerA,
		Expire:     expireTime,
	})
	assert.NoError(t, err)
	assert.Equal(t, tryLock.Success, true)
	unlock, _ := comp.Unlock(&lock.UnlockRequest{
		ResourceId: resouseId,
		LockOwner:  lockOwerA,
	})
	assert.NoError(t, err)
	assert.Equal(t, unlock.Status, lock.SUCCESS)

}

// A lock ,B unlock
func TestZookeeperLock_ALock_BUnlock(t *testing.T) {

	comp := NewZookeeperLock(log.DefaultLogger)
	comp.Init(cfg)

	//mock
	ctrl := gomock.NewController(t)
	unlockConn := NewMockZKConnection(ctrl)
	lockConn := NewMockZKConnection(ctrl)
	factory := NewMockConnectionFactory(ctrl)
	path := "/" + resouseId
	factory.EXPECT().NewConnection(time.Duration(expireTime)*time.Second, comp.metadata).Return(lockConn, nil).Times(2)

	lockConn.EXPECT().Create(path, []byte(lockOwerA), int32(zk.FlagEphemeral), zk.WorldACL(zk.PermAll)).Return("", nil).Times(1)
	lockConn.EXPECT().Close().Return().Times(1)

	unlockConn.EXPECT().Get(path).Return([]byte(lockOwerA), &zk.Stat{Version: 123}, nil).Times(1)
	unlockConn.EXPECT().Delete(path, int32(123)).Return(nil).Times(1)

	comp.unlockConn = unlockConn
	comp.factory = factory

	tryLock, err := comp.TryLock(&lock.TryLockRequest{
		ResourceId: resouseId,
		LockOwner:  lockOwerA,
		Expire:     expireTime,
	})
	assert.NoError(t, err)
	assert.Equal(t, tryLock.Success, true)
	unlock, err := comp.Unlock(&lock.UnlockRequest{
		ResourceId: resouseId,
		LockOwner:  lockOwerB,
	})
	assert.NoError(t, err)
	assert.Equal(t, unlock.Status, lock.LOCK_BELONG_TO_OTHERS)

}

// A lock , B lock ,A unlock ,B lock,B unlock
func TestZookeeperLock_ALock_BLock_AUnlock_BLock_BUnlock(t *testing.T) {

	comp := NewZookeeperLock(log.DefaultLogger)
	comp.Init(cfg)

	//mock
	ctrl := gomock.NewController(t)
	unlockConn := NewMockZKConnection(ctrl)
	lockConn := NewMockZKConnection(ctrl)
	factory := NewMockConnectionFactory(ctrl)
	path := "/" + resouseId

	factory.EXPECT().NewConnection(time.Duration(expireTime)*time.Second, comp.metadata).Return(lockConn, nil).Times(3)

	lockConn.EXPECT().Create(path, []byte(lockOwerA), int32(zk.FlagEphemeral), zk.WorldACL(zk.PermAll)).Return("", nil).Times(1)
	lockConn.EXPECT().Create(path, []byte(lockOwerB), int32(zk.FlagEphemeral), zk.WorldACL(zk.PermAll)).Return("", zk.ErrNodeExists).Times(1)
	lockConn.EXPECT().Create(path, []byte(lockOwerB), int32(zk.FlagEphemeral), zk.WorldACL(zk.PermAll)).Return("", nil).Times(1)
	lockConn.EXPECT().Close().Return().Times(5)

	unlockConn.EXPECT().Get(path).Return([]byte(lockOwerA), &zk.Stat{Version: 123}, nil).Times(1)
	unlockConn.EXPECT().Get(path).Return([]byte(lockOwerB), &zk.Stat{Version: 124}, nil).Times(1)
	unlockConn.EXPECT().Delete(path, int32(123)).Return(nil).Times(2)
	unlockConn.EXPECT().Delete(path, int32(124)).Return(nil).Times(2)

	comp.unlockConn = unlockConn
	comp.factory = factory

	//A lock
	tryLock, err := comp.TryLock(&lock.TryLockRequest{
		ResourceId: resouseId,
		LockOwner:  lockOwerA,
		Expire:     expireTime,
	})
	assert.NoError(t, err)
	assert.Equal(t, true, tryLock.Success)
	//B lock
	tryLock, err = comp.TryLock(&lock.TryLockRequest{
		ResourceId: resouseId,
		LockOwner:  lockOwerB,
		Expire:     expireTime,
	})
	assert.NoError(t, err)
	assert.Equal(t, false, tryLock.Success)
	//A unlock
	unlock, _ := comp.Unlock(&lock.UnlockRequest{
		ResourceId: resouseId,
		LockOwner:  lockOwerA,
	})
	assert.NoError(t, err)
	assert.Equal(t, lock.SUCCESS, unlock.Status)

	//B lock
	tryLock, err = comp.TryLock(&lock.TryLockRequest{
		ResourceId: resouseId,
		LockOwner:  lockOwerB,
		Expire:     expireTime,
	})
	assert.NoError(t, err)
	assert.Equal(t, true, tryLock.Success)

	//B unlock
	unlock, _ = comp.Unlock(&lock.UnlockRequest{
		ResourceId: resouseId,
		LockOwner:  lockOwerB,
	})
	assert.NoError(t, err)
	assert.Equal(t, lock.SUCCESS, unlock.Status)
}
