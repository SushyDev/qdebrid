handle 204 (no content) when there's nothing in the users RD


queue for adding / select
sometimes we add a torrent but reach the api rate limit and the select fails, this leaves unselected torrent. we need to queue with retry this
