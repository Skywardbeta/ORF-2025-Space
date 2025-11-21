# Squidの設定

## SSL Bumpについて
- 今回の参照した記事
：https://tech-mmmm.blogspot.com/2021/09/squidssl-bumphttps-ssl.html

<p>&nbsp;</p>
<!--空行-->

### SSL Bumpとは
　HTTPSでは、ブラウザとWebサーバ間の通信が暗号化されるため、当然ではあるが、通信の詳細な情報を第三者が確認することはできない。

　しかし、近年はフィッシング詐欺などの悪意のあるサイトであっても、SSLサーバ証明書を導入しHTTPS通信とすることで、一見問題がないようなサイトと偽装することも多くなっている。そのような危険な通信を可視化することためには、**一度HTTPSの通信を復号し、セキュリティチェックを実施したうえで、再度暗号化する**ことが定番の手法となっている。

※ ただし、復号化と暗号化処理が増えることから、専用のネットワーク機器ではない場合はスループットが極端に落ちるため、導入時は注意する。

<!--空行-->
<p>&nbsp;</p>

### SSL Bumpの仕組み
- i. SSL Bumpでは、クライアントから見るとSquidが宛先のサーバーであるかのようにふるまい、クライアントのSSL通信をSquidが一旦終端する。（クライアントがHTTPSでウェブサイトにアクセスしようとすると、まずSquidに接続する。）
- ii. Squidは、クライアントからのリクエストを受け取ると、クライアントがアクセスしようとしている本来のオリジンサーバー（例: `www.example.com`）に対してSSL接続を確立する。
- iii. オリジンサーバーは、Squidに対して自身の**正規の**サーバー証明書を提示する。
- iv. Squidはオリジンサーバーから受け取った情報を基に、クライアント向けに**新しいSSL証明書を動的に「生成」する**。この新しく生成された証明書は、アクセス先のドメイン名（例: `www.example.com`）を含むSquidが生成した公開鍵(ACによって署名済み)と、**Squid自身のルート証明書(とペアとなる秘密鍵)で署名された**ACの公開鍵（証明書）から構成される。（※なお、ACはSquid自身で偽造している。）
- v. Squidはこの動的に生成した証明書をクライアントに提示する。クライアントはSquidから受け取った証明書を検証する。この際、クライアントのトラストストアにSquidのルート証明書がインポートされていれば、Squidが発行した証明書を信頼し、サーバーの真正性が確認される。よって、SSL接続が確立される。

<!--空行-->
<p>&nbsp;</p>

※参照：HTTPSの通信について

[ディフィー・ヘルマン鍵共有](https://qiita.com/yasushi-jp/items/bbd28049f8fa295d8e25)
以前はRSA等の公開鍵暗号方式で、共通鍵を暗号化して渡すこともあったのですが、暗号化されているとはいえ共通鍵が第三者に見られることは避けたい、といった背景があります。
ディフィー・ヘルマン鍵共有だと、通信で流すのは公開パラメータとDH公開鍵と第三者に見られても問題がない値のみなので、鍵交換プロトコルとして使用されています。
最近では「楕円曲線ディフィー・ヘルマン鍵共有（Elliptic curve Diffie–Hellman key exchange, ECDH）」も使用されているようです。

<p>&nbsp;</p>
<!--空行-->
<p>&nbsp;</p>

## Squidでの設定手順
### 1. SSL Bump用の公開パラメータの生成

SSL Bumpでは、Squidが都度SSL証明書を発行する。このクライアント⇔Squid間で実施するSSL通信の鍵交換の際に使用する鍵情報となる「Diffie-Hellmanパラメータ (DHパラメータ)」を事前に作成する。

（DHパラメータはopensslコマンドにて生成でき、処理に数分を要する。ファイル名はbump_dhparam.pemとして作成する。）

```
$ cd /etc/squid/
$ sudo openssl dhparam -outform PEM -out bump_dhparam.pem 2048

$ ll
total 1020
-rw-r--r-- 1 root root    424 Nov  5 20:56 bump_dhparam.pem
drwxr-xr-x 2 root root   4096 Nov  3 01:36 conf.d
-rw-r--r-- 1 root root   1800 Oct 28 00:28 errorpage.css
-rw-r--r-- 1 root root 343256 Nov  3 02:26 squid.conf
-rw-r--r-- 1 root root 343185 Oct 28 00:28 squid.conf_20251103
-rw-r--r-- 1 root root 343256 Nov  5 20:55 squid.conf_20251105
```


「p」：大きな安全な素数（数千ビット規模）

「g」：原始根（生成元）

これらのセットをまとめたものが「DHパラメータファイル（bump_dhparam.pem）」。

この仕組みでは：

- 双方がランダムな秘密値 a, b を生成

- 公開値 A = g^a mod p, B = g^b mod p を交換

- それぞれ K = B^a mod p = A^b mod p という同じ共通鍵を計算

(そのため、サーバーの真正性の担保と、機密性の担保ではロジックが別)

<!--空行-->
<p>&nbsp;</p>

### 2. SSL Bump用のサーバ証明書
次に、SSL通信の際に使用するSSLサーバ証明書の秘密鍵と公開鍵のペアを作成する。秘密鍵のファイル名はbump.key、公開鍵のファイル名はbump.crtとし、opensslコマンドで作成を行う。(RSA)

→　ルート証明書の生成

```
$ cd /etc/squid/
$ sudo openssl req -new -newkey rsa:2048 -days 3650 -nodes -x509 -keyout bump.key -out bump.crt

$ ll
total 1028
-rw-r--r-- 1 root root   1245 Nov  5 21:00 bump.crt
-rw------- 1 root root   1704 Nov  5 21:00 bump.key
-rw-r--r-- 1 root root    424 Nov  5 20:56 bump_dhparam.pem
drwxr-xr-x 2 root root   4096 Nov  3 01:36 conf.d
-rw-r--r-- 1 root root   1800 Oct 28 00:28 errorpage.css
-rw-r--r-- 1 root root 343256 Nov  3 02:26 squid.conf
-rw-r--r-- 1 root root 343185 Oct 28 00:28 squid.conf_20251103
-rw-r--r-- 1 root root 343256 Nov  5 20:55 squid.conf_20251105
```
<!--空行-->
<p>&nbsp;</p>

### 3. SSL証明書補完用DBの作成
前述の通り、SSL Bumpでは、SquidがクライアントとSSL通信をするため、都度（Squid自身の）サーバー証明書を発行する。(作成した証明書情報を保管するためのDBを作成する。)

※OSによってコマンドが一部ことなる（Ubuntuでの実行時）

→　キャッシュ
```
$ sudo mkdir -p /var/lib/squid
$ sudo rm -rf /var/lib/squid/ssl_db

$ sudo apt install squid-openssl

# -c　オプションでcreate.
# -s でDBの保存先を指定.
$ sudo /usr/lib/squid/security_file_certgen -c -s /var/lib/squid/ssl_db -M 20MB
$ sudo chown -R proxy:proxy /var/lib/squid
```

<!--空行-->
<p>&nbsp;</p>

### 4. squid.confに設定追加
SquidでSSL Bumpを有効化するため、squid.confに設定を追加する。

以下を追加
```
# 証明書作成のコマンドを指定。-sオプションでSSL保管先のDBを指定する。-MオプションでDBに保管するSSL証明書の最大サイズを指定する。
# 2927行目付近
sslcrtd_program /usr/lib64/squid/security_file_certgen -s /var/lib/squid/ssl_db -M 20MB
```
```
# 証明書の検証にてエラーがあってもすべて許可する。
# 2803行付近
sslproxy_cert_error allow all
```
```
# SSL Bumpのアクションをstep1についてpeekとし、それ以外のstep2、step3についてはbumpに設定する。(詳細は公式サイトを参照すること。)
# 1216行目付近
acl step1 at_step SslBump1
ssl_bump peek step1
# 2785行目付近
ssl_bump bump all
```

また、SSL Bumpの場合は以下のように多数の設定を行う。
```
# 2120行目付近
http_port 8083 tcpkeepalive=60,30,3 ssl-bump generate-host-certificates=on dynamic_cert_mem_cache_size=20MB tls-cert=/etc/squid/bump.crt tls-key=/etc/squid/bump.key cipher=HIGH:MEDIUM:!LOW:!RC4:!SEED:!IDEA:!3DES:!MD5:!EXP:!PSK:!DSS options=NO_TLSv1,NO_SSLv3,SINGLE_DH_USE,SINGLE_ECDH_USE tls-dh=prime256v1:/etc/squid/bump_dhparam.pem
```

<table style="width:100%; border-collapse:collapse; table-layout:fixed;">
  <colgroup>
    <col style="width:32%;">
    <col style="width:68%;">
  </colgroup>
  <thead>
    <tr>
      <th style="padding:10px; border:1px solid #ccc; text-align:left;">設定値</th>
      <th style="padding:10px; border:1px solid #ccc; text-align:left;">説明</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td style="padding:10px; border:1px solid #ccc;">`http_port 8080`</td>
      <td style="padding:10px; border:1px solid #ccc;">Squidの待ち受けポートを8080番に設定。</td>
    </tr>
    <tr>
      <td style="padding:10px; border:1px solid #ccc;">`tcpkeepalive=60,30,3`</td>
      <td style="padding:10px; border:1px solid #ccc;">通信がアイドルとなった際のキープアライブの設定。左から順に、アイドルとみなす経過時間(秒)、アイドル中の監視間隔(秒)、タイムアウトとみなす回数(秒)となる。今回の設定の場合は、60秒経過後から30秒 x 3回通信が確認できなかった場合、通信の切断を行う。</td>
    </tr>
    <tr>
      <td style="padding:10px; border:1px solid #ccc;">`ssl-bump`</td>
      <td style="padding:10px; border:1px solid #ccc;">SSL Bumpを使用する。これによりクライアントから受信したHTTPS通信が一度復号化される。</td>
    </tr>
    <tr>
      <td style="padding:10px; border:1px solid #ccc;">`generate-host-certificates=on`</td>
      <td style="padding:10px; border:1px solid #ccc;">SSLサーバー証明書を動的に作成する。</td>
    </tr>
    <tr>
      <td style="padding:10px; border:1px solid #ccc;">`dynamic_cert_mem_cache_size=20MB`</td>
      <td style="padding:10px; border:1px solid #ccc;">証明書作成時に使用するキャッシュのサイズ。デフォルト4MB。</td>
    </tr>
    <tr>
      <td style="padding:10px; border:1px solid #ccc;">`tls-cert=/etc/squid/bump.crt`</td>
      <td style="padding:10px; border:1px solid #ccc;">SSLサーバ証明書のパスを指定。</td>
    </tr>
    <tr>
      <td style="padding:10px; border:1px solid #ccc;">`tls-key=/etc/squid/bump.key`</td>
      <td style="padding:10px; border:1px solid #ccc;">SSLサーバ証明書の秘密鍵のパスを指定。</td>
    </tr>
    <tr>
      <td style="padding:10px; border:1px solid #ccc;">`cipher=HIGH:MEDIUM:!LOW:!RC4:!SEED:!IDEA:!3DES:!MD5:!EXP:!PSK:!DSS`</td>
      <td style="padding:10px; border:1px solid #ccc;">暗号化スイートの設定。</td>
    </tr>
    <tr>
      <td style="padding:10px; border:1px solid #ccc;">`options=NO_TLSv1,NO_SSLv3,SINGLE_DH_USE,SINGLE_ECDH_USE`</td>
      <td style="padding:10px; border:1px solid #ccc;">オプションの設定。TLSv1禁止、SSLv3禁止、一時的なDHキーの使用をするためオプションを記載。</td>
    </tr>
    <tr>
      <td style="padding:10px; border:1px solid #ccc;">`tls-dh=prime256v1:/etc/squid/bump_dhparam.pem`</td>
      <td style="padding:10px; border:1px solid #ccc;">DHパラメータの指定。使用する曲線とファイルパスを指定。</td>
    </tr>
  </tbody>
</table>

<!--空行-->
<p>&nbsp;</p>

squid.confの設定例
```
# ACLs（Access Control List）
# acl(キーワード) acl名（任意）src(通信方向) IPアドレス範囲
acl localnet src 192.168.33.0/24
acl localnet src 192.168.11.0/24

acl SSL_ports port 443
acl Safe_ports port 80          # http
acl Safe_ports port 21          # ftp
acl Safe_ports port 443         # https
acl Safe_ports port 70          # gopher
acl Safe_ports port 210         # wais
acl Safe_ports port 1025-65535  # unregistered ports
acl Safe_ports port 280         # http-mgmt
acl Safe_ports port 488         # gss-http
acl Safe_ports port 591         # filemaker
acl Safe_ports port 777         # multiling http
# 1355行目付近
acl CONNECT method CONNECT

# SSL Bump
sslcrtd_program /usr/lib/squid/security_file_certgen -s /var/lib/squid/ssl_db -M 20MB
sslproxy_cert_error allow all
acl step1 at_step SslBump1
ssl_bump peek step1
ssl_bump bump all

# Access rules
http_access deny !Safe_ports
http_access deny CONNECT !SSL_ports

http_access allow localhost manager
http_access deny manager

# 許可する通信の種類　allow 通信を許可するacl名
http_access allow localnet
http_access allow localhost
http_access deny all

# Squid Access Pport
http_port 8083 tcpkeepalive=60,30,3 ssl-bump generate-host-certificates=on dynamic_cert_mem_cache_size=20MB tls-cert=/etc/squid/bump.crt tls-key=/etc/squid/bump.key cipher=HIGH:MEDIUM:!LOW:!RC4:!SEED:!IDEA:!3DES:!MD5:!EXP:!PSK:!DSS options=NO_TLSv1,NO_SSLv3,SINGLE_DH_USE,SINGLE_ECDH_USE tls-dh=prime256v1:/etc/squid/bump_dhparam.pem

# Cache directory
# 3750行目付近
cache_dir ufs /var/spool/squid 100 16 256

# Core dump directory
coredump_dir /var/spool/squid

# Cache settings
refresh_pattern ^ftp:           1440    20%     10080
refresh_pattern ^gopher:        1440    0%      1440
refresh_pattern -i (/cgi-bin/|\?) 0     0%      0
refresh_pattern .               0       20%     4320
```

<!--空行-->
<p>&nbsp;</p>

設定を反映させるためsquidを再起動
```
$ sudo systemctl restart squid
```
<!--空行-->
<p>&nbsp;</p>
ログの確認
```
$ sudo tail -f /var/log/squid/access.log
```

<!--空行-->
<p>&nbsp;</p>

### 5. OSにルート証明書を登録

先ほど作成したSquidの証明書となるbump.crtをエクスポートして、インポート対象のOSに配置する。

<!--空行-->
<p>&nbsp;</p>

### 6. 上位プロキシ（転送先）の設定
squidを使って、SSL終端できるキャッシュサーバを立てたが、次の設定でmemoryのサイズを指定できる。

```
# 3635行目付近
cache_mem 256 MB
```

localhostの3000番ポートにリクエストを転送する設定を追加。
```
# 3082行目付近
cache_peer localhost            parent    3000     0  no-query originserver
```

### 7. リクエストをを捻じ曲げる
あらゆるリクエストを`www.google.com/`に転送したい
```
# 5070行目付近
url_rewrite_program /usr/local/squid/rewrite.sh
url_rewrite_access allow all
url_rewrite_children 5 startup=5 idle=5
```

/usr/local/squid/rewrite.shを次のように編集
```
#!/bin/bash
while read url
do
  echo "www.google.com"
donei
```
```
sudo chmod +x /usr/local/squid/rewrite.sh
```
