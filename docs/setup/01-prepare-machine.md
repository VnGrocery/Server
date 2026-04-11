# 01. Prepare Machine

## Cài sẵn những thứ này
- Git
- Docker Engine + Docker Compose
- Go
- `make`
- `curl`
- `jq` là tốt, nhưng không bắt buộc

## Kiểm tra nhanh
```bash
git --version
docker --version
docker compose version
go version
make --version
```

## Thư mục bạn sẽ dùng
Ví dụ:

```bash
mkdir -p ~/workspace
cd ~/workspace
```

## Nếu Docker báo lỗi quyền
Bạn có thể phải dùng `sudo` trước các lệnh `docker` hoặc `docker compose`.

Ví dụ:
```bash
sudo docker ps
sudo docker compose version
```
