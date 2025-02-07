# 現在の値を取得してファイルに出力

```
aws-ss-cli dump -id <secret id> -f .env
```

# ファイルの値をsecret managerに書き出す

```
aws-ss-cli change -id <secret id> -f .env
```

# 直前の変更を戻す

```
aws-ss-cli revert -id <secret id>
```
