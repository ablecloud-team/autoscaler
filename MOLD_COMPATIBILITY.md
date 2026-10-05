# Mold Kubernetes 구성 요소 검증 후보

기준: Kubernetes cluster-autoscaler-1.34.5 cb2123ed13148c38fbdb1ede42d8e99761ee2909. 내부 HMAC-SHA256 및 저장소 네임스페이스를 유지하면서 원본 변경을 병합했습니다.

Origin Actions에서 계약/기능 테스트 후 amd64 바이너리, 커밋 고정 이미지, Go 모듈 목록, provenance와 docker archive를 생성합니다. 공식 배포와 Origin 후보를 구분하며 ISO는 후보의 실제 digest와 source SHA를 lock에 기록합니다.

Kubernetes 1.34용 원본 1.34.5 기준입니다. 1.35/1.36 인터페이스를 이 버전에 무조건 이식하지 않습니다. 목록 조회, size 증가 및 nodeids 삭제를 사용하는 CKS 계약을 유지합니다. 지원되지 않는 zero-size, atomic scaling, nodegroup 생성/삭제는 성공으로 표시하지 않습니다.

client는 Mold 값 인코딩/서명 규칙과 오류/async 응답 검증을 적용합니다. 공유 고정 벡터는 `cluster-autoscaler/cloudprovider/cloudstack/service/testdata/mold-signing.json`입니다. 인증정보는 테스트 전용입니다.
