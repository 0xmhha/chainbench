# Regression Test Result

> 출처: Confluence [Regression Test Result](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2634843002) (페이지 ID 2634843002, 버전 1, 최종 수정 2026-04-20)  
> 상위 페이지: Regression Test  
> 가져온 날짜: 2026-09-18 (본문은 Confluence markdown 변환 결과를 그대로 옮김. 원본 상단의 목차 매크로는 생략)

---

## 테스트 케이스 

-  <https://wemade.atlassian.net/wiki/x/GQD2mg> 


## 테스트 환경

- 폐쇄망 서버

| IP | 역할 | 타입 |
| --- | --- | --- |
| 172.21.132.1 \~ .7 | Validator Node | 블록 생성·합의 |
| 172.21.132.8 | EN Node (Snap Sync) | 동기화 노드 |
| 172.21.132.9 | EN Node (Snap Sync) | 동기화 노드 |
| 172.21.132.10 | EN Node (Snap Sync) | 동기화 노드 |
| 172.21.132.11 | EN Node (Full Sync) | 동기화 노드 |
| 172.21.132.12 | EN Node (Full Sync) | 동기화 노드 |
| 172.21.132.13 | EN Node (Full Sync) | 동기화 노드 |
| 172.21.132.14 | EN Node (Full Sync) | 동기화 노드 |
| 172.21.132.15 | PN (EN + Bootnode) | P2P 연결 기준점 |


## 테스트 사전 작업

- gstable build
    - go v1.23.12
    - branch : dev
    - 기준 커밋 : <https://github.com/stable-net/go-stablenet/commit/e5a6e9e14c1e1d225c341b55798f31cd07b0bfcd> 
        - 추가 수정사항 : 테스트 환경 구성을 위해 기준 커밋에 별도 수정 적용
- pn (bootnode) 정보
    - 

```
--bootnodes enode://ed62b9e5410eb4ae3e1f08b305c9b623faefda632d1f435934b888fda8dbc131f2894b9e9fa0107945df49b4f3e2d054589b8f5c10aa6e543d5d6aa179a9d31f@172.21.132.15:30301
```
