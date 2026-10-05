# Mold Kubernetes 구성 요소 검증 후보

기준: Kubernetes cluster-autoscaler-1.34.5 cb2123ed13148c38fbdb1ede42d8e99761ee2909. 내부 HMAC-SHA256 및 저장소 네임스페이스를 유지하면서 원본 변경을 병합했습니다.

Origin Actions에서 계약/기능 테스트 후 amd64 바이너리, 커밋 고정 이미지, Go 모듈 목록, provenance와 docker archive를 생성합니다. 공식 배포와 Origin 후보를 구분하며 ISO는 후보의 실제 digest와 source SHA를 lock에 기록합니다.

Kubernetes 1.34용 원본 1.34.5 기준입니다. 1.35/1.36 인터페이스를 이 버전에 무조건 이식하지 않습니다. 목록 조회, size 증가 및 nodeids 삭제를 사용하는 CKS 계약을 유지합니다. 지원되지 않는 zero-size, atomic scaling, nodegroup 생성/삭제는 성공으로 표시하지 않습니다.

client는 Mold 값 인코딩/서명 규칙과 오류/async 응답 검증을 적용합니다. 공유 고정 벡터는 `cluster-autoscaler/cloudprovider/cloudstack/service/testdata/mold-signing.json`입니다. 인증정보는 테스트 전용입니다.

## Minor별 추가 검증

Kubernetes 1.34.12, 1.35.9, 1.36.5, 1.37.1 요청에 따라 4개 minor를 개별 빌드합니다. `mold/baselines.json`의 원본 commit에 공통 Mold client 및 고정 signing vector를 적용하여 각 minor 인터페이스의 CloudStack tests를 실행한 후 바이너리를 만듭니다. 다른 minor의 인터페이스를 복사하지 않습니다.

1.37용 안정 AutoScaler 릴리즈는 아직 없습니다. 원본 1.37 개발 코드 commit을 고정한 검증 후보로 구분합니다. source SHA는 내부 patch 저장소 commit이고 binary_source_sha는 해당 minor의 원본 코드 commit입니다. client 파일과 테스트의 SHA256, Go build info 및 이미지 label을 같이 검증해야 합니다.
