# backend-heartbeat

Beep backend records and makes available the last seen times of users.

**To run this service securely means to run it behind traefik forwarding auth to `backend-auth`**

## Environment variables

Supply environment variables by either exporting them or editing ```.env```.

| ENV | Description | Default |
| ---- | ----------- | ------- |
| LISTEN | Host and port number to listen on | :8080 |
| REDIS | Host and port of redis | :6379 |

## Status Codes

The system supports arbitrary status codes. However, in the interest of standardisation, a system of codes (based on Skype statuses) is listed here:

| Code | Description |
| ---- | ----------- |
| 0 | Active |
| 1 | Away |
| 2 | Do not disturb |
| 3 | Invisible |

## API

### Subscribe User

```
GET /subscribe/:userid/client/:clientid
```

Subscribe to a user. Every time a user pings this service, the time will be sent to all subscribed users. Upon subscription, if it exists, the last cached ping of the target user will be pushed immediately to the stream.

```js
const es = new EventSource(`${host}/subscribe/${user}/client/${device}`);
es.onmessage = (e) => {
  const timestamp = e.data;
  // Do whatever with the ping data
};
```

Ping data:

```json
{
  "time": "<UTC epoch timestamp>",
  "status": "<status code>"
}
```

#### URL Params

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| userid | String | Target user's ID. | ✓ |
| clientid | String | Target user's device's ID. | ✓ |

#### Success Response (200 OK)

An [EventSource](https://developer.mozilla.org/en-US/docs/Web/API/EventSource) stream.

---

### Ping Server

```
POST /ping
```

Ping the server.

#### Required headers

| Name | Description |
| ---- | ----------- |
| X-User-Claim | Stringified user claim, populated by `backend-auth` called by `traefik` |

#### Body

| Name | Type | Description |
| ---- | ---- | ----------- |
| status | String | Status code |

#### Success Response (200 OK)

Empty body.

#### Errors

| Code | Description |
| ---- | ----------- |
| 400 | Invalid user claims header. |
