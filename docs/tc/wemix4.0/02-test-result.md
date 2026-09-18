# [WEMIX 4.0] 테스트 결과

> 출처: Confluence [[WEMIX 4.0] 테스트 결과](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2693038270) (페이지 ID 2693038270, 버전 1, 최종 수정 2026-05-18)  
> 상위 페이지: [WEMIX4.0] Test  
> 가져온 날짜: 2026-09-18 (본문은 Confluence markdown 변환 결과를 그대로 옮김)

---

## 테스트 케이스

- [\[WEMIX4.0\] Test Case](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2636711400/WEMIX4.0+Test)

## 테스트 환경

- 폐쇄망 서버

| IP | 역할 | 타입 |
| --- | --- | --- |
| 172.21.132.1 | WEMIX3.0 BP | 블록 생성·합의 ( \~99 블록) |
| 172.21.132.2 | WEMIX3.0 BP(0\~99),WEMIX4.0 (100\~) | 블록 생성·합의 ( \~99 ),데이터 마이그레이션 테스트 (3.0 데이터 + 4.0 바이너리) |
| 172.21.132.3 \~ .9 | 4.0 Validator Node | 블록 생성·합의 (100 \~ ), |
| 172.21.132.10 \~ .13 | EN Node (Full Sync) | 동기화 노드 |
| 172.21.132.14 | EN Node (Snap Sync) | 동기화 노드 |
| 172.21.132.15 | PN (EN + Bootnode) | P2P 연결 기준점 |


## 테스트 사전 작업

- gwemix3 build
    - go v1.19
    - branch : master
    - 기준 커밋 : <https://github.com/wemixarchive/go-wemix/commit/a9fc03fea184fc9d99f956b74226d6e2772164f6> 
        - 추가 수정사항 : 테스트 환경 구성을 위해 기준 커밋에 별도 수정 적용
- gwemix4 build
    - go 1.23.12
    - branch : dev
    - 기준 커밋 : <https://github.com/wemixarchive/go-wbft/commit/3d7a0a5367dcf5e4ab86fc8475b764468e815316>  
        - 추가 수정사항 : 테스트 환경 구성을 위해 기준 커밋에 별도 수정 적용
- pn (bootnode) 정보
    - 

```
--bootnodes enode://ed62b9e5410eb4ae3e1f08b305c9b623faefda632d1f435934b888fda8dbc131f2894b9e9fa0107945df49b4f3e2d054589b8f5c10aa6e543d5d6aa179a9d31f@172.21.132.15:30301
```
