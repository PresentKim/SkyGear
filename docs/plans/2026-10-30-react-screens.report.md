# superlite build report

projection: fixed forecast saved at lint 2026-10-30T05:21:08 (plan edited since)

                         Projected    Actual  Delta
wall-clock (min)              16.4      18.1  +1.7 (+10%)
agent cost (USD, +rework)      4.81      6.08  +1.27 (+26%)
task-minutes                  65.1      47.0  -18.1 (-28%)
agents                          28        28  +0 (+0%)
cost split: agents $5.86, rework $0.21, coordinator $103.87, total $109.95
concurrency: peak 9 at once (projected 10); critical path actual T13 T19 T23 T26; projected T1 > T9 > T20 > T25 > T23 > T26 (13.9 min)
model mix: haiku 1 tasks $0.02, sonnet 26 tasks $6.05, coordinator opus $103.87
quality: tests 115 passed, 0 failed; first-time pass 1/1 (100%); rework dispatches 2; open incidents 25

task  model      pred  actual  delta             cost
T10   sonnet      2.8     0.8  -2.0 (-71%)     $0.155
T21   sonnet      2.8     0.9  -1.9 (-69%)     $0.152
T12   sonnet      2.9     1.3  -1.6 (-57%)     $0.225
T0    haiku       1.5     3.1  +1.6 (+108%)    $0.024
T20   sonnet      3.6     2.2  -1.4 (-40%)     $0.328
(+22 smaller misses; `sl report --json` lists all)

agent cost table:
| agent | model | wall_min | tokens | usd |
| --- | --- | --- | --- | --- |
| a0cd711bfc27cb441 | sonnet | 29888978.1879 | 0 | 0.0 |
| a0e83577f8cb78955 | sonnet | 1.1687 | 247576 | 0.2037 |
| a43f71a017f72c955 | sonnet | 4.6918 | 1123647 | 0.3943 |
| a456fd2efe6ca8854 | sonnet | 0.9374 | 285200 | 0.1977 |
| a4b0c9f6cc6fef85d | sonnet | 3.4779 | 417663 | 0.2171 |
| a573bcdab3112cf38 | sonnet | 29888975.6101 | 0 | 0.0 |
| a58017e91699227a9 | sonnet | 1.409 | 286190 | 0.2513 |
| a59f792af85f45add | sonnet | 1.2609 | 336335 | 0.2252 |
| a5b6676eb1daf8f2c | sonnet | 1.9626 | 879448 | 0.3555 |
| a5d6e628ec7ffe7b0 | sonnet | 1.5494 | 404677 | 0.2807 |
| a6340e5f761de2391 | sonnet | 2.0285 | 1095879 | 0.4183 |
| a79671dde1b287f5e | sonnet | 9.0618 | 1336633 | 0.5809 |
| a7dbef6b335947cfb | sonnet | 3.5056 | 366127 | 0.2498 |
| a8939dbb2f1ecaa46 | sonnet | 5.786 | 720132 | 0.3551 |
| a8ece0a46bfb7706e | sonnet | 5.139 | 2409359 | 0.6979 |
| aa0eedeac1d281651 | haiku | 3.1052 | 522511 | 0.0244 |
| aa47c0a4b1803f8aa | sonnet | 3.3357 | 597390 | 0.3341 |
| aa4c84bbfca71f646 | sonnet | 0.5203 | 128206 | 0.0603 |
| aa77bad64f8e2b01d | sonnet | 6.1752 | 1189110 | 0.5313 |
| ab07282ca33536e1e | sonnet | 0.8265 | 226997 | 0.175 |
| ab6e1bc31a9fd6f8d | sonnet | 1.1821 | 391491 | 0.1543 |
| abc4b58d528f7a1dd | opus | 0.4562 | 33891 | 0.1774 |
| ac30d3c2832bc7b04 | sonnet | 29888982.7505 | 0 | 0.0 |
| ad3f4360ee6631285 | sonnet | 1.2869 | 287381 | 0.2272 |
| adce1485e75f7f541 | sonnet | 29888976.9481 | 0 | 0.0 |
| aee7e681e749b7291 | sonnet | 1.6905 | 346746 | 0.2635 |
| aeff8a0f556f413f4 | sonnet | 2.4876 | 1117639 | 0.3535 |
| af498da421df6ce48 | sonnet | 6.3972 | 569380 | 0.2979 |
| totals |  | 119555982.9386 | 15319608 | 7.0264 |
cost source: est

takeaway: ran 10% slower than projected; cost 26% over; biggest miss T10
