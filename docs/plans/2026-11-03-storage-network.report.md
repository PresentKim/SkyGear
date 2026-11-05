# superlite build report

projection: fixed forecast saved at lint 2026-11-03T11:45:10 (plan edited since)

                         Projected    Actual  Delta
wall-clock (min)              38.5      30.6  -8.0 (-21%)
agent cost (USD, +rework)      2.48      6.70  +4.23 (+171%)
task-minutes                  89.9      39.9  -50.0 (-56%)
agents                          14        26  +12 (+86%)
cost split: agents $5.59, rework $1.11, coordinator $14.38, total $21.08
concurrency: peak 5 at once (projected 4); critical path actual T5; projected T0 > T1 > T7 > T8 > T9 > T13 (38.5 min)
model mix: haiku 1 tasks $0.20, sonnet 11 tasks $6.50, coordinator opus $14.38
quality: tests 42 passed, 0 failed; first-time pass n/a; rework dispatches 3; open incidents 68

task  model      pred  actual  delta             cost
T4    sonnet      6.4     1.1  -5.3 (-82%)     $0.142
T1    sonnet      6.4     1.1  -5.3 (-82%)     $0.124
T7    sonnet      6.4     1.7  -4.7 (-74%)     $0.237
T2    sonnet      6.4     1.8  -4.6 (-72%)     $0.203
T13   sonnet      6.4     1.8  -4.6 (-71%)     $0.458
(+7 smaller misses; `sl report --json` lists all)

agent cost table:
| agent | model | wall_min | tokens | usd |
| --- | --- | --- | --- | --- |
| a156e3e00e04d7dc6 | sonnet | 8.2043 | 2556668 | 0.9761 |
| a1e5919d503414209 | sonnet | 1.5712 | 283976 | 0.1978 |
| a27cc2ad13ede6c70 | sonnet | 29895126.2052 | 0 | 0.0 |
| a355e4d15f6cc6788 | sonnet | 3.0349 | 805294 | 0.3492 |
| a3ceb0632f74f98f5 | sonnet | 5.8463 | 1806327 | 0.7304 |
| a4432fa5435296670 | sonnet | 29895112.2545 | 0 | 0.0 |
| a5c85d3969b7ca44d | sonnet | 3.1654 | 681818 | 0.287 |
| a6538b5c344e6d17a | sonnet | 29895132.8017 | 0 | 0.0 |
| a7a23edf004fe9da6 | sonnet | 29895124.0672 | 0 | 0.0 |
| a7e0bcac131feebf5 | sonnet | 29895120.7811 | 0 | 0.0 |
| a8ce73f7597d74101 | sonnet | 29895128.7232 | 0 | 0.0 |
| a906c34e95577d122 | sonnet | 29895121.7357 | 0 | 0.0 |
| a914a43b8b14d9807 | sonnet | 13.036 | 1714890 | 0.7581 |
| a9772063eabbaf243 | sonnet | 29895124.822 | 0 | 0.0 |
| a988d37f6a35ec5e0 | sonnet | 8.1174 | 4828123 | 1.2878 |
| a9b8c54275eb6e958 | opus | 2.3382 | 500436 | 0.474 |
| ab3c4710249c41f71 | sonnet | 29895114.5697 | 0 | 0.0 |
| acb837481a1636767 | opus | 0.4546 | 34079 | 0.19 |
| acf83be0786616b26 | sonnet | 3.9233 | 1040418 | 0.475 |
| ad045c6bb8078c95e | sonnet | 29895111.7291 | 0 | 0.0 |
| adc701f2e817f49af | sonnet | 29895113.6344 | 0 | 0.0 |
| ae885f683c2d51d17 | sonnet | 29895118.9614 | 0 | 0.0 |
| aef2974c8b231d6c0 | sonnet | 29895129.6219 | 0 | 0.0 |
| af786d3b58cba3957 | haiku | 0.322 | 103293 | 0.0042 |
| af8c21c10f1043894 | sonnet | 29895112.8723 | 0 | 0.0 |
| afe219f758d12a78b | sonnet | 16.9604 | 2577618 | 1.1634 |
| totals |  | 418531759.7534 | 16932940 | 6.893 |
cost source: est

takeaway: ran 21% faster than projected; cost 171% over; biggest miss T4
