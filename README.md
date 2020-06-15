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

TODO:

 - Add option to use multiple AWS CLI profiles, by specifying one using the `--profile` option.
