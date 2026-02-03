---
name: issue-resolver
description: this is github issue resolver. when you asked to deal with github issue, please follow this instruction.
---

github issueに取り組むときの手順

1.  `gh issue list`コマンドで`https://github.com/Tetsu-is/social-media-scaling`のissueを取得します。
2.  取得したissueの中から、どのissueに取り組むかユーザーに選択してもらいます。
3.  選択されたissueの詳細（説明、コメント）を`gh issue view <issue-id>`で読み込み、要件を正確に把握します。
4.  issueの解決策について具体的な設計・実装計画を立て、ユーザーに提示して承認を得ます。
5.  承認後、issue番号を含んだブランチ名を決定します (例: `issue/123-fix-bug`)。
6.  `git worktree add ./tmp/worktrees/<branch-name> -b <branch-name>` のように`git worktree`を使い、隔離された作業ディレクトリを作成します。これにより、ユーザーの現在のブランチに影響を与えることなく安全に作業できます。
7.  作成したworktreeのディレクトリ (`./tmp/worktrees/<branch-name>`) に移動し、そこでコーディング、テスト、デバッグのサイクルを回します。
8.  実装が完了したら、変更をadd, commitし、`git push origin <branch-name>`でリモートにpushします。
9.  `gh pr create`コマンドでPull Requestを作成し、issueを紐付けます。
10. PRがマージされるか、作業が不要になった場合は、`git worktree remove ./tmp/worktrees/<branch-name>`でworktreeを削除し、`git branch -d <branch-name>`でローカルブランチも削除してクリーンな状態を保ちます。