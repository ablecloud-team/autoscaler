# Mold Kubernetes 구성 요소 검증 후보

기준: Kubernetes cluster-autoscaler-1.34.5 cb2123ed13148c38fbdb1ede42d8e99761ee2909. 내부 HMAC-SHA256 및 저장소 네임스페이스를 유지하면서 원본 변경을 병합했습니다.

Origin Actions에서 계약/기능 테스트 후 amd64 바이너리, 커밋 고정 이미지, Go 모듈 목록, provenance와 docker archive를 생성합니다. 공식 배포와 Origin 후보를 구분하며 ISO는 후보의 실제 digest와 source SHA를 lock에 기록합니다.

Kubernetes 1.34용 원본 1.34.5 기준입니다. 1.35/1.36 인터페이스를 이 버전에 무조건 이식하지 않습니다. 목록 조회, size 증가 및 nodeids 삭제를 사용하는 CKS 계약을 유지합니다. 지원되지 않는 zero-size, atomic scaling, nodegroup 생성/삭제는 성공으로 표시하지 않습니다.

client는 Mold 값 인코딩/서명 규칙과 오류/async 응답 검증을 적용합니다. 공유 고정 벡터는 `cluster-autoscaler/cloudprovider/cloudstack/service/testdata/mold-signing.json`입니다. 인증정보는 테스트 전용입니다.

## Minor별 추가 검증

Kubernetes 1.34.12, 1.35.9, 1.36.5, 1.37.1 요청에 따라 4개 minor를 개별 빌드합니다. `mold/baselines.json`의 원본 commit에 공통 Mold client 및 고정 signing vector를 적용하여 각 minor 인터페이스의 CloudStack tests를 실행한 후 바이너리를 만듭니다. 다른 minor의 인터페이스를 복사하지 않습니다.

1.37용 안정 AutoScaler 릴리즈는 아직 없습니다. 원본 1.37 개발 코드 commit을 고정한 검증 후보로 구분합니다. source SHA는 내부 patch 저장소 commit이고 binary_source_sha는 해당 minor의 원본 코드 commit입니다. client 파일과 테스트의 SHA256, Go build info 및 이미지 label을 같이 검증해야 합니다.

## 전체 저장소 라이선스 검사

License Check는 현재 HEAD의 추적 파일 전체를 Git archive로 분리하여 검사합니다.
Apache RAT 0.18로 소스 헤더를 검사하고, 헤더가 없는 기존 문서·구성·생성 SDK는
mold/license-coverage.json의 정확한 파일 경로와 SHA256을 대조합니다.
각 파일에 적용되는 기존 프로젝트/외부 라이브러리 LICENSE 파일과 그 SHA256도 검증합니다.
외부 소스를 일괄 Apache 라이선스로 변경하지 않습니다. 경로 glob으로 외부 소스 전체를 제외하지도 않습니다.

기존 OCI 소스는 Kubernetes 저장소의 Apache LICENSE와 기존 Oracle 저작권을 보존합니다.
Volcengine 생성 SDK에는 공식 저장소 be628166a0bada0efa7646ce6d0e9e484f0af163의
LICENSE.txt 및 NOTICE.txt 원문을 추가했으며 기존 코드·저작권은 수정하지 않았습니다.
나머지 외부 라이브러리는 각 디렉터리의 기존 라이선스 원문을 사용합니다.

예외 대상 파일 또는 라이선스가 변경되면 검사는 실패합니다. 새 소스는 인식 가능한 헤더를
포함해야 하며, 새 외부 라이브러리는 실제 원본 라이선스를 확인한 뒤 검토된 매핑을 추가해야 합니다.
자동으로 예외 목록을 갱신하거나 라이선스를 추측하는 단계는 CI에 없습니다.
검사 산출물은 RAT 원문 보고서, 모든 추적 파일 해시 및 라이선스 상속 검증 결과입니다.
