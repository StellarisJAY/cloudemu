package service

import (
	"context"
	"testing"
	"time"

	"github.com/StellarisJAY/cloudemu/internal/control-plane/contract"
	"github.com/StellarisJAY/cloudemu/internal/control-plane/model"
	"github.com/StellarisJAY/cloudemu/internal/pkg/apperror"

	"github.com/google/uuid"
)

// ---- 测试桩：嵌入接口，仅覆盖被测方法用到的方法 ----

type stubRoomRepo struct {
	contract.RoomRepo
	room *model.Room
}

func (s *stubRoomRepo) ByID(ctx context.Context, id uuid.UUID) (*model.Room, error) {
	return s.room, nil
}

type stubRoomPlayerRepo struct {
	contract.RoomPlayerRepo
	player *model.RoomPlayer
}

func (s *stubRoomPlayerRepo) ByRoomAndUser(ctx context.Context, roomID, userID uuid.UUID) (*model.RoomPlayer, error) {
	return s.player, nil
}

type stubSaveStateRepo struct {
	contract.SaveStateRepo
	byID      *model.SaveState
	created   *model.SaveState
	list      []model.SaveState
	latest    *model.SaveState
	renamedID uuid.UUID
	renamedTo string
	deletedID uuid.UUID
}

func (s *stubSaveStateRepo) Create(ctx context.Context, ss *model.SaveState) error {
	s.created = ss
	return nil
}

func (s *stubSaveStateRepo) ByID(ctx context.Context, id uuid.UUID) (*model.SaveState, error) {
	return s.byID, nil
}

func (s *stubSaveStateRepo) ListByRoom(ctx context.Context, roomID uuid.UUID) ([]model.SaveState, error) {
	return s.list, nil
}

func (s *stubSaveStateRepo) ListByRoomRom(ctx context.Context, roomID uuid.UUID, emulatorType string, romID uuid.UUID) ([]model.SaveState, error) {
	return s.list, nil
}

func (s *stubSaveStateRepo) LatestByRoomRom(ctx context.Context, roomID uuid.UUID, emulatorType string, romID uuid.UUID) (*model.SaveState, error) {
	return s.latest, nil
}

func (s *stubSaveStateRepo) Rename(ctx context.Context, id uuid.UUID, name string) error {
	s.renamedID = id
	s.renamedTo = name
	return nil
}

func (s *stubSaveStateRepo) Delete(ctx context.Context, id uuid.UUID) error {
	s.deletedID = id
	return nil
}

type stubMinio struct {
	contract.MinioFunc
	removedPath string
}

func (s *stubMinio) PresignedPutURL(ctx context.Context, bucket, path string, expiry time.Duration) (string, error) {
	return "http://minio/put", nil
}

func (s *stubMinio) PresignedGetURL(ctx context.Context, bucket, path string, expiry time.Duration) (string, error) {
	return "http://minio/get", nil
}

func (s *stubMinio) RemoveFile(ctx context.Context, bucket, path string) error {
	s.removedPath = path
	return nil
}

type stubWorkerClient struct {
	contract.WorkerClient
	saveSize   int64
	loadCalled bool
	saveCalled bool
}

func (s *stubWorkerClient) SaveState(ctx context.Context, workerAddr string, roomID, saveStateID uuid.UUID, uploadURL string) (int64, error) {
	s.saveCalled = true
	return s.saveSize, nil
}

func (s *stubWorkerClient) LoadState(ctx context.Context, workerAddr string, roomID, saveStateID uuid.UUID, downloadURL string) error {
	s.loadCalled = true
	return nil
}

// ---- SaveState 测试 ----

func TestRoomService_SaveState(t *testing.T) {
	hostID := uuid.Must(uuid.NewV7())
	otherID := uuid.Must(uuid.NewV7())
	roomID := uuid.Must(uuid.NewV7())
	romID := uuid.Must(uuid.NewV7())

	playingRoom := func() *model.Room {
		return &model.Room{ID: roomID, HostID: hostID, EmulatorType: "nes", RomID: &romID, Status: 1, WorkerAddr: "1.2.3.4:9090"}
	}

	tests := []struct {
		name    string
		caller  uuid.UUID
		room    *model.Room
		wantErr error
	}{
		{"房间不存在", hostID, nil, apperror.ErrRoomNotExist},
		{"非房主", otherID, playingRoom(), apperror.ErrNotRoomHost},
		{"非playing", hostID, &model.Room{ID: roomID, HostID: hostID, EmulatorType: "nes", RomID: &romID, Status: 0, WorkerAddr: "x"}, apperror.ErrRoomNotPlaying},
		{"未选ROM", hostID, &model.Room{ID: roomID, HostID: hostID, EmulatorType: "nes", RomID: nil, Status: 1, WorkerAddr: "x"}, apperror.ErrRomNotSelected},
		{"无Worker", hostID, &model.Room{ID: roomID, HostID: hostID, EmulatorType: "nes", RomID: &romID, Status: 1, WorkerAddr: ""}, apperror.ErrWorkerUnavailable},
		{"正常", hostID, playingRoom(), nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ssRepo := &stubSaveStateRepo{}
			wc := &stubWorkerClient{saveSize: 4096}
			svc := &RoomService{
				roomRepo:      &stubRoomRepo{room: tt.room},
				saveStateRepo: ssRepo,
				minioFunc:     &stubMinio{},
				workerClient:  wc,
				bucket:        "cloudemu",
			}

			ss, err := svc.SaveState(context.Background(), tt.caller, roomID)
			if err != tt.wantErr {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil {
				if ss == nil || ssRepo.created == nil {
					t.Fatal("正常路径应落库存档记录")
				}
				if ss.Size != 4096 || ss.RomID != romID || ss.EmulatorType != "nes" || ss.RoomID != roomID {
					t.Errorf("存档记录字段不正确: %+v", ss)
				}
				if !wc.saveCalled {
					t.Error("应调用 Worker.SaveState")
				}
			}
		})
	}
}

// ---- LoadState 测试 ----

func TestRoomService_LoadState(t *testing.T) {
	hostID := uuid.Must(uuid.NewV7())
	roomID := uuid.Must(uuid.NewV7())
	romID := uuid.Must(uuid.NewV7())
	saveID := uuid.Must(uuid.NewV7())

	room := &model.Room{ID: roomID, HostID: hostID, EmulatorType: "nes", RomID: &romID, Status: 1, WorkerAddr: "1.2.3.4:9090"}
	matchSave := &model.SaveState{ID: saveID, RoomID: roomID, EmulatorType: "nes", RomID: romID, MinioPath: "savestate/x.dat"}

	otherRoom := uuid.Must(uuid.NewV7())
	otherRom := uuid.Must(uuid.NewV7())

	tests := []struct {
		name    string
		save    *model.SaveState
		wantErr error
	}{
		{"存档不存在", nil, apperror.ErrSaveStateNotExist},
		{"房间不匹配", &model.SaveState{ID: saveID, RoomID: otherRoom, EmulatorType: "nes", RomID: romID}, apperror.ErrSaveStateMismatch},
		{"机种不匹配", &model.SaveState{ID: saveID, RoomID: roomID, EmulatorType: "gb", RomID: romID}, apperror.ErrSaveStateMismatch},
		{"ROM不匹配", &model.SaveState{ID: saveID, RoomID: roomID, EmulatorType: "nes", RomID: otherRom}, apperror.ErrSaveStateMismatch},
		{"三要素匹配", matchSave, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wc := &stubWorkerClient{}
			svc := &RoomService{
				roomRepo:      &stubRoomRepo{room: room},
				saveStateRepo: &stubSaveStateRepo{byID: tt.save},
				minioFunc:     &stubMinio{},
				workerClient:  wc,
				bucket:        "cloudemu",
			}

			rid := roomID
			sid := saveID
			err := svc.LoadState(context.Background(), hostID, contract.LoadStateReq{RoomID: &rid, SaveStateID: &sid})
			if err != tt.wantErr {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && !wc.loadCalled {
				t.Error("匹配时应调用 Worker.LoadState")
			}
			if tt.wantErr != nil && wc.loadCalled {
				t.Error("不匹配时不应调用 Worker.LoadState")
			}
		})
	}
}

// ---- ListSaveStates 测试 ----

func TestRoomService_ListSaveStates(t *testing.T) {
	userID := uuid.Must(uuid.NewV7())
	roomID := uuid.Must(uuid.NewV7())

	t.Run("非房间成员", func(t *testing.T) {
		svc := &RoomService{
			roomRepo:       &stubRoomRepo{room: &model.Room{ID: roomID}},
			roomPlayerRepo: &stubRoomPlayerRepo{player: nil},
			saveStateRepo:  &stubSaveStateRepo{},
		}
		_, err := svc.ListSaveStates(context.Background(), userID, roomID)
		if err != apperror.ErrNotInRoom {
			t.Fatalf("err = %v, want ErrNotInRoom", err)
		}
	})

	t.Run("成员可查（仅返回当前ROM/机种匹配）", func(t *testing.T) {
		romID := uuid.Must(uuid.NewV7())
		svc := &RoomService{
			roomRepo:       &stubRoomRepo{room: &model.Room{ID: roomID, EmulatorType: "nes", RomID: &romID}},
			roomPlayerRepo: &stubRoomPlayerRepo{player: &model.RoomPlayer{RoomID: roomID, UserID: userID}},
			saveStateRepo:  &stubSaveStateRepo{list: []model.SaveState{{ID: uuid.Must(uuid.NewV7())}}},
		}
		list, err := svc.ListSaveStates(context.Background(), userID, roomID)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if len(list) != 1 {
			t.Errorf("期望 1 条存档，得到 %d", len(list))
		}
	})

	t.Run("房间未选ROM返回空", func(t *testing.T) {
		svc := &RoomService{
			roomRepo:       &stubRoomRepo{room: &model.Room{ID: roomID, EmulatorType: "nes", RomID: nil}},
			roomPlayerRepo: &stubRoomPlayerRepo{player: &model.RoomPlayer{RoomID: roomID, UserID: userID}},
			saveStateRepo:  &stubSaveStateRepo{list: []model.SaveState{{ID: uuid.Must(uuid.NewV7())}}},
		}
		list, err := svc.ListSaveStates(context.Background(), userID, roomID)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if len(list) != 0 {
			t.Errorf("未选 ROM 应返回空，得到 %d", len(list))
		}
	})
}

// ---- LoadLatestState 测试 ----

func TestRoomService_LoadLatestState(t *testing.T) {
	hostID := uuid.Must(uuid.NewV7())
	otherID := uuid.Must(uuid.NewV7())
	roomID := uuid.Must(uuid.NewV7())
	romID := uuid.Must(uuid.NewV7())

	playingRoom := &model.Room{ID: roomID, HostID: hostID, EmulatorType: "nes", RomID: &romID, Status: 1, WorkerAddr: "1.2.3.4:9090"}
	latest := &model.SaveState{ID: uuid.Must(uuid.NewV7()), RoomID: roomID, EmulatorType: "nes", RomID: romID, MinioPath: "savestate/x.dat"}

	t.Run("非房主", func(t *testing.T) {
		svc := &RoomService{
			roomRepo:      &stubRoomRepo{room: playingRoom},
			saveStateRepo: &stubSaveStateRepo{latest: latest},
			minioFunc:     &stubMinio{},
			workerClient:  &stubWorkerClient{},
			bucket:        "cloudemu",
		}
		if err := svc.LoadLatestState(context.Background(), otherID, roomID); err != apperror.ErrNotRoomHost {
			t.Fatalf("err = %v, want ErrNotRoomHost", err)
		}
	})

	t.Run("无存档", func(t *testing.T) {
		svc := &RoomService{
			roomRepo:      &stubRoomRepo{room: playingRoom},
			saveStateRepo: &stubSaveStateRepo{latest: nil},
			minioFunc:     &stubMinio{},
			workerClient:  &stubWorkerClient{},
			bucket:        "cloudemu",
		}
		if err := svc.LoadLatestState(context.Background(), hostID, roomID); err != apperror.ErrSaveStateNotExist {
			t.Fatalf("err = %v, want ErrSaveStateNotExist", err)
		}
	})

	t.Run("正常加载最新", func(t *testing.T) {
		wc := &stubWorkerClient{}
		svc := &RoomService{
			roomRepo:      &stubRoomRepo{room: playingRoom},
			saveStateRepo: &stubSaveStateRepo{latest: latest},
			minioFunc:     &stubMinio{},
			workerClient:  wc,
			bucket:        "cloudemu",
		}
		if err := svc.LoadLatestState(context.Background(), hostID, roomID); err != nil {
			t.Fatalf("err = %v", err)
		}
		if !wc.loadCalled {
			t.Error("期望调用 Worker LoadState")
		}
	})
}

// ---- RenameSaveState 测试 ----

func TestRoomService_RenameSaveState(t *testing.T) {
	hostID := uuid.Must(uuid.NewV7())
	otherID := uuid.Must(uuid.NewV7())
	roomID := uuid.Must(uuid.NewV7())
	ssID := uuid.Must(uuid.NewV7())

	room := &model.Room{ID: roomID, HostID: hostID}
	ss := &model.SaveState{ID: ssID, RoomID: roomID}
	req := func() contract.RenameSaveStateReq {
		return contract.RenameSaveStateReq{RoomID: &roomID, SaveStateID: &ssID, Name: "关卡3"}
	}

	t.Run("非房主", func(t *testing.T) {
		svc := &RoomService{roomRepo: &stubRoomRepo{room: room}, saveStateRepo: &stubSaveStateRepo{byID: ss}}
		if err := svc.RenameSaveState(context.Background(), otherID, req()); err != apperror.ErrNotRoomHost {
			t.Fatalf("err = %v, want ErrNotRoomHost", err)
		}
	})

	t.Run("存档不存在", func(t *testing.T) {
		svc := &RoomService{roomRepo: &stubRoomRepo{room: room}, saveStateRepo: &stubSaveStateRepo{byID: nil}}
		if err := svc.RenameSaveState(context.Background(), hostID, req()); err != apperror.ErrSaveStateNotExist {
			t.Fatalf("err = %v, want ErrSaveStateNotExist", err)
		}
	})

	t.Run("正常重命名", func(t *testing.T) {
		ssRepo := &stubSaveStateRepo{byID: ss}
		svc := &RoomService{roomRepo: &stubRoomRepo{room: room}, saveStateRepo: ssRepo}
		if err := svc.RenameSaveState(context.Background(), hostID, req()); err != nil {
			t.Fatalf("err = %v", err)
		}
		if ssRepo.renamedID != ssID || ssRepo.renamedTo != "关卡3" {
			t.Errorf("重命名参数错误：id=%v name=%q", ssRepo.renamedID, ssRepo.renamedTo)
		}
	})
}

// ---- DeleteSaveState 测试 ----

func TestRoomService_DeleteSaveState(t *testing.T) {
	hostID := uuid.Must(uuid.NewV7())
	otherID := uuid.Must(uuid.NewV7())
	roomID := uuid.Must(uuid.NewV7())
	ssID := uuid.Must(uuid.NewV7())

	room := &model.Room{ID: roomID, HostID: hostID}
	ss := &model.SaveState{ID: ssID, RoomID: roomID, MinioPath: "savestate/room/x.dat"}
	req := contract.DeleteSaveStateReq{RoomID: &roomID, SaveStateID: &ssID}

	t.Run("非房主", func(t *testing.T) {
		svc := &RoomService{roomRepo: &stubRoomRepo{room: room}, saveStateRepo: &stubSaveStateRepo{byID: ss}, minioFunc: &stubMinio{}}
		if err := svc.DeleteSaveState(context.Background(), otherID, req); err != apperror.ErrNotRoomHost {
			t.Fatalf("err = %v, want ErrNotRoomHost", err)
		}
	})

	t.Run("存档不存在", func(t *testing.T) {
		svc := &RoomService{roomRepo: &stubRoomRepo{room: room}, saveStateRepo: &stubSaveStateRepo{byID: nil}, minioFunc: &stubMinio{}}
		if err := svc.DeleteSaveState(context.Background(), hostID, req); err != apperror.ErrSaveStateNotExist {
			t.Fatalf("err = %v, want ErrSaveStateNotExist", err)
		}
	})

	t.Run("正常删除（清MinIO+删记录）", func(t *testing.T) {
		ssRepo := &stubSaveStateRepo{byID: ss}
		mn := &stubMinio{}
		svc := &RoomService{roomRepo: &stubRoomRepo{room: room}, saveStateRepo: ssRepo, minioFunc: mn}
		if err := svc.DeleteSaveState(context.Background(), hostID, req); err != nil {
			t.Fatalf("err = %v", err)
		}
		if mn.removedPath != ss.MinioPath {
			t.Errorf("未删除 MinIO 对象，removedPath=%q", mn.removedPath)
		}
		if ssRepo.deletedID != ssID {
			t.Errorf("未删除记录，deletedID=%v", ssRepo.deletedID)
		}
	})
}
