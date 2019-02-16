# backend-heartbeat

Beep backend records and makes available the last seen times of users.

## API

### Subscribe User

```
GET /subscribe/:userid/client/:clientid
```

Subscribe to a user. Every time a user pings this service, the time will be sent to all subscribed users.

```js
const es = new EventSource(`${host}/subscribe/${user}/client/${device}`);
es.onmessage = (e) => {
  const timestamp = e.data;
  // Do whatever with the timestamp
};
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
POST /ping/:userid/client/:clientid
```

Ping the server.

#### URL Params

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| userid | String | User's ID. | ✓ |
| clientid | String | User's device's ID. | ✓ |

#### Success Response (200 OK)

Empty body.
