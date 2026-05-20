# Luong 07: Enum hoa trace event va validate timeline

Trang thai: `todo`

## Van de

Trace event hien chi yeu cau `type` khong rong. Timeline truy xuat nguon goc co the bi nhap sai type, sai thu tu, hoac thieu event cot loi ma backend khong canh bao.

## Muc tieu

- Trace event type co taxonomy ro rang.
- Backend reject type khong hop le.
- Co validate toi thieu ve timeline va event bat buoc.
- Trace event co audit/integrity neu muc tieu la bang chung truy xuat nguon goc.

## Taxonomy de xuat

- `origin`
- `slaughter`
- `packaging`
- `shipping`
- `received`
- `storage_check`
- `freshness_check`
- `recall`
- `disposal`

## Timeline rules de xuat

- `occurredAt` khong duoc qua xa tuong lai.
- `received` khong nen truoc `shipping`.
- `packaging` khong nen truoc `slaughter` neu ca hai cung co.
- `recall` duoc tao bat ky luc nao nhung nen cap nhat batch status `recalled`.
- `storage_check` va `freshness_check` co the lap lai.

## Pham vi code du kien

- `internal/service/traceability/service.go`
- `internal/service/traceability/service_test.go`
- Co the can `ProductBatchRepository.Save` neu recall event cap nhat batch status.
- Audit service neu can signed event log cho trace event.

## Backend tasks

- Them constants cho trace event types.
- Them `validateTraceEventType`.
- Validate `occurredAt`.
- Khi tao `recall`:
  - cap nhat batch status `recalled`, hoac ghi TODO neu chua lam trong luong nay
- Xem xet tao audit event `trace_event.created`.
- Sort list trace events theo `OccurredAt` neu repository chua dam bao.

## Tests can co

- Tao event voi type hop le thanh cong.
- Reject type khong hop le.
- Reject occurredAt qua xa tuong lai.
- Recall event cap nhat batch status neu implement.
- Public list chi tra active events va sort dung.
- `go test ./internal/service/traceability`
- `go test ./...`

## Da lam

- Chua lam.

## Test da chay

- Chua chay.

## Commit

De xuat:

```text
Validate trace event taxonomy
```
