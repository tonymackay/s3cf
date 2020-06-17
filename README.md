# s3cf
A go program that syncs a static website to an S3 bucket and purges URLs stored in Cloudflare's edge cache. 

This program uses the AWS CLI to execute the `sync` command using the `--size-only` and `--delete` options. The `--delete` option makes sure files that have been deleted locally are removed from the bucket and the `--size-only` option prevents local files from being copied to the S3 bucket if they have been regenerated before syncing (for example, using `hugo build`).

Since it's possible for locally modified files to be missed by the `--size-only` option, the program will also do a second pass comparing the MD5 of each local file with the remote ETag value. If they don't match, the local file will overwrite the file stored in the S3 bucket.

## Prerequisites
The program requires the AWS CLI and needs credentials configured.

## Environment Variables


The following variables are required in order to purge modified URLS from Cloudflare's edge cache.

```
export S3CF_CF_API_KEY=<cloudflare_api_key>
export S3CF_CF_API_EMAIL=<cloudflare_email>
export S3CF_CF_API_ZONE=<cloudflare_zone_id>
export S3CF_CF_BASE_URL=<https://mywebsite.com
```

## Usage
The following command will sync the contents of the `www` folder to the S3 bucket named `mybucketname` and purge the cache of any modified URLS. 

```
s3cf www s3://mybucketname
```

Output: 

```
Syncing www with s3://mybucketname
(dryrun) upload: www/hello-world/index.html to s3://mybucketname/hello-world/index.html
(dryrun) upload: www/index.html to s3://mybucketname/index.html
Purging URLs from CloudFlare Cache
Purged: [https://mywebsite.com/hello-world/, https://mywebsite.com/]
```

If you need to specify a specific profile in the aws credentials file. You can use the `-profile` option. For example:

```
s3cf -profile development www s3://mybucketname
```

## License
MIT License

Copyright (c) 2020 Tony Mackay

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
