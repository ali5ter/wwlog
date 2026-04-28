# Weight Watchers API Reference

Unofficial reference for the WW REST API, documented through live reverse-engineering
against a real US account. Covers all reachable endpoints as of April 2026.

> **Stability:** This is an undocumented private API. Endpoints and field names
> may change without notice.

---

## Base URLs

All URLs are parameterised by a regional TLD.

| TLD      | Region          |
|----------|-----------------|
| `com`    | United States   |
| `ca`     | Canada          |
| `com.au` | Australia       |
| `co.uk`  | United Kingdom  |

```
Auth base:  https://auth.weightwatchers.{tld}
Data base:  https://cmx.weightwatchers.{tld}
```

---

## Authentication

WW uses a two-step flow to produce a JWT (`id_token`) used for all data requests.

---

### Step 1 — Credentials → Session token

**`POST https://auth.weightwatchers.{tld}/login-apis/v1/authenticate`**

#### Request

```http
POST /login-apis/v1/authenticate HTTP/1.1
Content-Type: application/json
```

```json
{
  "username":        "user@example.com",
  "password":        "secret",
  "rememberMe":      false,
  "usernameEncoded": false,
  "retry":           false
}
```

#### Response `200 OK`

```json
{
  "data": {
    "tokenId": "AQIC5wM2LY..."
  }
}
```

| Field          | Type   | Meaning                                              |
|----------------|--------|------------------------------------------------------|
| `data.tokenId` | string | Opaque short-lived session token; pass to Step 2 as a cookie |

Non-200 means bad credentials.

#### Examples

```bash
TOKEN_ID=$(curl -s -X POST \
  "https://auth.weightwatchers.com/login-apis/v1/authenticate" \
  -H "Content-Type: application/json" \
  -d '{"username":"user@example.com","password":"secret","rememberMe":false,"usernameEncoded":false,"retry":false}' \
  | jq -r '.data.tokenId')
```

```python
import requests

resp = requests.post(
    "https://auth.weightwatchers.com/login-apis/v1/authenticate",
    json={"username": "user@example.com", "password": "secret",
          "rememberMe": False, "usernameEncoded": False, "retry": False},
)
resp.raise_for_status()
token_id = resp.json()["data"]["tokenId"]
```

---

### Step 2 — Session token → JWT

**`GET https://auth.weightwatchers.{tld}/openam/oauth2/authorize`**

Exchanges the session token for a signed JWT via an OAuth2 implicit-style flow.
The JWT arrives in the **URL fragment** of a `302` redirect — the client must
**not** follow the redirect.

#### Query parameters

| Parameter       | Value                                                   |
|-----------------|---------------------------------------------------------|
| `response_type` | `id_token`                                              |
| `client_id`     | `webCMX`                                                |
| `redirect_uri`  | `https://cmx.weightwatchers.{tld}/auth` (URL-encoded)   |
| `nonce`         | Any random string (prevents replay)                     |

#### Cookie

| Name      | Value                        |
|-----------|------------------------------|
| `wwAuth2` | `tokenId` from Step 1        |

#### Response `302 Found`

```
Location: https://cmx.weightwatchers.{tld}/auth#id_token=eyJ...&token_type=Bearer&...
```

Parse the URL fragment (`#…`) as a query string. The `id_token` value is the JWT.

#### JWT payload

Standard JWT. Relevant claims:

| Claim | Meaning                                      |
|-------|----------------------------------------------|
| `exp` | Expiry as Unix timestamp                     |
| `sub` | WW member UUID                               |

Tokens expire after roughly 1 hour. Re-authenticate before expiry.

#### Examples

```bash
REDIRECT_URI="https://cmx.weightwatchers.com/auth"
NONCE=$(openssl rand -base64 12)

LOCATION=$(curl -s -o /dev/null -w "%{redirect_url}" \
  --cookie "wwAuth2=${TOKEN_ID}" \
  "https://auth.weightwatchers.com/openam/oauth2/authorize\
?response_type=id_token&client_id=webCMX\
&redirect_uri=$(python3 -c "import urllib.parse,sys; print(urllib.parse.quote(sys.argv[1]))" "$REDIRECT_URI")\
&nonce=${NONCE}")

JWT=$(echo "$LOCATION" | grep -oP 'id_token=\K[^&]+')
```

```python
import urllib.parse, secrets, requests

redirect_uri = "https://cmx.weightwatchers.com/auth"
nonce = secrets.token_urlsafe(16)

resp = requests.get(
    "https://auth.weightwatchers.com/openam/oauth2/authorize",
    params={"response_type": "id_token", "client_id": "webCMX",
            "redirect_uri": redirect_uri, "nonce": nonce},
    cookies={"wwAuth2": token_id},
    allow_redirects=False,
)
assert resp.status_code == 302
fragment = urllib.parse.parse_qs(urllib.parse.urlparse(resp.headers["Location"]).fragment)
jwt = fragment["id_token"][0]
```

```javascript
import fetch from "node-fetch";
import { randomBytes } from "node:crypto";

const redirectUri = "https://cmx.weightwatchers.com/auth";
const nonce = randomBytes(12).toString("base64url");
const params = new URLSearchParams({
  response_type: "id_token", client_id: "webCMX",
  redirect_uri: redirectUri, nonce,
});

const resp = await fetch(
  `https://auth.weightwatchers.com/openam/oauth2/authorize?${params}`,
  { headers: { Cookie: `wwAuth2=${tokenId}` }, redirect: "manual" },
);
const location = resp.headers.get("location");
const fragment = new URLSearchParams(new URL(location).hash.slice(1));
const jwt = fragment.get("id_token");
```

---

## Data requests

All data endpoints require:

```http
Authorization: Bearer <id_token>
```

Both `~` and the explicit member UUID are accepted wherever `~` appears in paths.

---

### Member Profile

**`GET https://cmx.weightwatchers.{tld}/api/v2/cmx/members/~/profile`**

Returns full member account data: demographics, plan settings, goals, and
preferences. Also accessible at `/api/v1/cmx/members/~/profile` (identical response).

#### Example

```bash
curl -s \
  -H "Authorization: Bearer ${JWT}" \
  "https://cmx.weightwatchers.com/api/v2/cmx/members/~/profile" \
  | jq .
```

#### Response fields

| Field                          | Type     | Meaning                                                       |
|--------------------------------|----------|---------------------------------------------------------------|
| `id`                           | string   | Member UUID — same as JWT `sub` claim                        |
| `uuid`                         | string   | Alias for `id`                                                |
| `username`                     | string   | WW username                                                   |
| `wwEmail`                      | string   | WW account email                                              |
| `firstName` / `lastName`       | string   | Name                                                          |
| `birthDate`                    | string   | `YYYY-MM-DD`                                                  |
| `gender`                       | string   | `"M"` or `"F"`                                               |
| `height`                       | integer  | Height in centimetres                                         |
| `bmi`                          | float    | Current BMI                                                   |
| `weight`                       | float    | Current body weight                                           |
| `units`                        | string   | Weight unit for `weight`: `"lbs"` or `"kg"`                  |
| `startWeight`                  | float    | Weight at programme enrolment                                 |
| `startWeightDate`              | string   | `YYYY-MM-DD` of enrolment weight                             |
| `goalWeight`                   | float    | Target body weight                                            |
| `goalWeightUnits`              | string   | Unit for `goalWeight`                                        |
| `pointsProgram`                | string   | Active points plan name — currently `"maxpointsSimple"`      |
| `trackingMode`                 | string   | `"count"` — points tracking mode                             |
| `swappingMode`                 | string   | Points swapping setting, e.g. `"noSwapping"`                 |
| `weightLossMode`               | string   | `"lose"` or `"maintenance"`                                   |
| `weighInDay`                   | string   | Day of the week for weekly weigh-in, e.g. `"monday"`         |
| `rollover`                     | boolean  | Whether unused daily points roll over to weekly allowance     |
| `dptAdjustment`                | integer  | Manual daily points target adjustment                         |
| `wpaAdjustment`                | integer  | Manual weekly points allowance adjustment                     |
| `enrollmentDate`               | string   | ISO 8601 timestamp of programme enrolment                    |
| `registrationDate`             | string   | ISO 8601 timestamp of account creation                       |
| `zpfMix`                       | string[] | Zero-point food categories enabled for this member           |
| `preferredNutrients`           | string[] | Nutrients the member tracks (display order in the app)       |
| `hasDiabetes`                  | boolean  | Diabetes programme flag                                       |
| `has_glp`                      | boolean  | GLP-1 programme participant flag                              |
| `entitlements`                 | string[] | Active subscription features, e.g. `"app"`, `"studio"`      |
| `countryCode` / `market`       | string   | Two-letter country code                                       |
| `timezone`                     | string   | UTC offset, e.g. `"-04:00"`                                  |
| `preferredHeightWeightUnits`   | string   | `"imperial"` or `"metric"`                                    |
| `sleepGoalSettings.goal`       | integer  | Sleep goal in minutes                                         |
| `activityGoalSettings.minutesGoal` | integer | Weekly activity goal in minutes                          |
| `waterSettings`                | object   | Water tracking: `enabled`, `servingSize`, `servingUnit`      |
| `maintenanceDPTRanges`         | object   | `min`, `max`, `base` daily points range for maintenance mode |
| `avatarUrl`                    | string   | Profile photo URL                                             |
| `additionalFields.profile.about.myWhy` | string | Member's stated motivation                          |
| `additionalFields.app.experience` | object | App version/feature flags                                 |
| `timestamp`                    | string   | ISO 8601 time the response was generated                     |

---

### My-Day — daily food log

**`GET https://cmx.weightwatchers.{tld}/api/v3/cmx/operations/composed/members/~/my-day/{date}`**

The core endpoint. Returns everything for one calendar day: all tracked food
entries with full embedded nutrition, daily and weekly points budgets, activity
points, body weight, and veggie servings. No secondary calls are needed —
nutrition is embedded in each food entry.

The response always includes **two top-level keys**: `today` (the requested date)
and `yesterday` (the day before). Both have identical structure.

#### Path parameter

| Parameter | Format       | Example      |
|-----------|--------------|--------------|
| `{date}`  | `YYYY-MM-DD` | `2026-04-20` |

#### Example

```bash
curl -s \
  -H "Authorization: Bearer ${JWT}" \
  "https://cmx.weightwatchers.com/api/v3/cmx/operations/composed/members/~/my-day/2026-04-20" \
  | jq .
```

```python
import requests

resp = requests.get(
    "https://cmx.weightwatchers.com/api/v3/cmx/operations/composed/members/~/my-day/2026-04-20",
    headers={"Authorization": f"Bearer {jwt}"},
)
resp.raise_for_status()
today = resp.json()["today"]
```

#### Response shape

```
{
  "today": {
    "pointsDetails": { ... },
    "trackedFoods": {
      "morning": [ <food entry>, ... ],
      "midday":  [ <food entry>, ... ],
      "evening": [ <food entry>, ... ],
      "anytime": [ <food entry>, ... ]
    }
  },
  "yesterday": {
    "pointsDetails": { ... },
    "trackedFoods": { ... }
  }
}
```

---

#### `pointsDetails`

Daily and weekly WW points budget, plus weight and veggie servings for the day.

| Field                              | Type    | Unit  | Meaning                                                          |
|------------------------------------|---------|-------|------------------------------------------------------------------|
| `localDate`                        | string  | —     | Date this data applies to (`YYYY-MM-DD`)                         |
| `plan`                             | string  | —     | Active points plan, e.g. `"maxpointsSimple"`                    |
| `trackingMode`                     | string  | —     | `"count"` — points tracking mode                                |
| `weightLossMode`                   | string  | —     | `"lose"` or `"maintenance"` for this day                        |
| `swappingMode`                     | string  | —     | Points swapping setting                                          |
| `dailyPointTarget`                 | float   | pts   | Daily points budget                                              |
| `dailyPointTargetAdjustment`       | float   | pts   | Manual adjustment applied to the daily target                   |
| `dailyPointsUsed`                  | float   | pts   | Points consumed from daily budget                               |
| `dailyPointsRemaining`             | float   | pts   | Daily budget remaining; can go negative                         |
| `dailyRolloverPointsEarned`        | float   | pts   | Daily points that rolled over from the previous day             |
| `weeklyPointAllowance`             | float   | pts   | Weekly flex allowance total                                     |
| `weeklyPointAllowanceAdjustment`   | float   | pts   | Manual adjustment to the weekly allowance                       |
| `weeklyPointAllowanceUsed`         | float   | pts   | Weekly flex points spent                                        |
| `weeklyPointAllowanceRemaining`    | float   | pts   | Weekly flex points still available                              |
| `totalPointsUsed`                  | float   | pts   | Combined daily + weekly points used                             |
| `dailyActivityPointsEarned`        | float   | pts   | FitPoints earned today                                          |
| `dailyActivityPointsUsed`          | float   | pts   | Activity points applied today                                   |
| `dailyActivityPointsRemaining`     | float   | pts   | Activity points not yet applied                                 |
| `weeklyActivityPointsEarned`       | float   | pts   | FitPoints earned this week                                      |
| `weeklyActivityPointsUsed`         | float   | pts   | Activity points applied this week                               |
| `weeklyActivityPointsRemaining`    | float   | pts   | Activity points remaining this week                             |
| `dailyVeggieServings`              | integer | count | Whole-number veggie/ZeroPoint servings logged                   |
| `dailyVeggieServingsPrecise`       | float   | count | Precise veggie servings count                                   |
| `morningAutoVeggies`               | float   | count | Veggie servings auto-detected in morning meal                   |
| `middayAutoVeggies`                | float   | count | Veggie servings auto-detected in midday meal                    |
| `eveningAutoVeggies`               | float   | count | Veggie servings auto-detected in evening meal                   |
| `anytimeAutoVeggies`               | float   | count | Veggie servings auto-detected in anytime meal                   |
| `manualVeggieDPTBonus`             | float   | pts   | Points bonus earned from manual veggie logging                  |
| `dailyWaterBonus`                  | float   | pts   | Points bonus earned from water tracking                         |
| `maxPointsBaselineDPT`             | float   | pts   | Baseline daily target before adjustments                        |
| `maxPointsBaselineWPA`             | float   | pts   | Baseline weekly allowance before adjustments                    |
| `weight`                           | float   | varies | Most recent logged body weight                                  |
| `weightUnit`                       | string  | —     | Weight unit: `"kgs"` or `"lbs"` *(note: plural `"kgs"`, not `"kg"`)* |
| `weightDate`                       | string  | —     | Date the weight was logged (`YYYY-MM-DD`)                       |
| `weighInDay`                       | string  | —     | Configured weekly weigh-in day, e.g. `"monday"`                |
| `zpfMix`                           | string[]| —     | Zero-point food categories active on this day                   |

> **Legacy fields:** `dailyPointPlusTarget`, `weeklyPointsPlusAllowance`,
> `smartPointsCountPrecise`, and similar `*Plus*` / `*SmartPoints*` fields are
> echoed from older plan schemas. They hold the same values as their current
> counterparts and can be ignored.

---

#### Food entry object

Each item in `morning`, `midday`, `evening`, or `anytime`.

| Field              | Type    | Meaning                                                              |
|--------------------|---------|----------------------------------------------------------------------|
| `entryId`          | string  | Unique ID for this log entry (UUID)                                 |
| `_id`              | string  | WW database ID for the food definition                              |
| `versionId`        | string  | Version ID of the food definition at time of tracking               |
| `portionId`        | string  | ID of the `defaultPortion` object used                             |
| `name`             | string  | Display food name                                                    |
| `_ingredientName`  | string  | Full product name including brand                                   |
| `_displayName`     | string  | Name shown in the app (often same as `_ingredientName`)             |
| `_servingDesc`     | string  | Pre-formatted serving description, e.g. `"1 cup(s)"`               |
| `portionName`      | string  | Portion unit label, e.g. `"cup(s)"`, `"oz"`, `"serving"`           |
| `portionSize`      | float   | Quantity of the tracked portion                                     |
| `quantity`         | float   | Same as `portionSize`; kept for legacy compatibility               |
| `sourceType`       | string  | Origin of the food — see [Source types](#source-types)             |
| `trackedDate`      | string  | Date logged (`YYYY-MM-DD`)                                          |
| `timeOfDay`        | string  | Meal period: `"morning"` / `"midday"` / `"evening"` / `"anytime"` |
| `eventAt`          | string  | ISO 8601 UTC timestamp of when the entry was created               |
| `isActive`         | boolean | `false` if the entry has been soft-deleted                         |
| `isZPF`            | boolean | Whether this food counts as zero points for the member              |
| `isGlpApproved`    | boolean | Whether this food is approved for GLP-1 programme participants      |
| `points`           | integer | Rounded points charged to the daily budget                         |
| `pointsPrecise`    | float   | Exact points charged to the daily budget                           |
| `proteinGrams`     | float   | Protein in grams (convenience field; duplicates nutrition data)    |
| `vegetableServings`| integer | Whole veggie servings this entry contributes                       |
| `glpServings`      | integer | GLP servings this entry contributes                                |
| `validationIssues` | array   | Warnings about data quality (usually empty)                        |
| `images`           | object  | Food images metadata (often empty)                                 |
| `nutritionData`    | object  | Always empty `{}`; nutrition comes from `defaultPortion.nutrition` |
| `pointsInfo`       | object  | Detailed points breakdown — see below                              |
| `defaultPortion`   | object  | Reference portion with nutrition data — see below                  |

> **Which points field to use for "how many points did this cost?":**
> Use `pointsPrecise` (float) or `points` (rounded integer) — these reflect the
> actual points charged. For ZPF foods `pointsPrecise` is `0`.
> `pointsInfo.maxPoints` is the *calculated* points value before ZPF status is
> applied — useful if you want to show "saved X points by eating ZPF".

---

#### `pointsInfo` object

| Field            | Type               | Meaning                                                        |
|------------------|--------------------|----------------------------------------------------------------|
| `maxPoints`      | float              | Calculated points before ZPF; equals `pointsPrecise` for non-ZPF |
| `maxPointsHigh`  | float              | High end of the points range (for foods with variable nutrition)|
| `maxPointsLow`   | float              | Low end of the points range                                    |
| `zpfComp`        | map<string, float> | ZPF category contributions that reduced the points value       |
| `amount`         | object             | `{ quantity: float, unit: string }` — tracked weight/volume   |
| `vegetableServings` | integer         | Veggie servings credited                                       |
| `glpServings`    | integer            | GLP servings credited                                          |
| `proteinGrams`   | float              | Protein grams used for points calculation                      |
| `_id`            | string             | Internal ID for this pointsInfo record                         |

> **`zpfComp`** maps ZPF category names (e.g. `"yogurt"`, `"poultry"`) to the
> fraction of points they zeroed out. For non-ZPF foods it shows minor
> contributions from ZPF ingredients inside recipes (e.g. `"wwpasta": 0.19`).

---

#### `defaultPortion` object

The canonical reference portion for the food. **Nutrition values are for one
unit of this reference portion** (`defaultPortion.size`), not for the tracked
amount. Scale by `portionSize / defaultPortion.size` to get values for the
actual serving logged.

| Field        | Type               | Meaning                                                    |
|--------------|--------------------|------------------------------------------------------------|
| `_id`        | string             | Same as `portionId` on the entry                           |
| `name`       | string             | Portion unit name                                          |
| `size`       | float              | Canonical portion size — the scaling denominator           |
| `weight`     | float              | Reference portion weight in grams                         |
| `weightType` | string             | Always `"g"`                                               |
| `default`    | boolean            | Whether this is the food's default portion                |
| `isActive`   | boolean            | Whether this portion definition is current               |
| `points`     | integer            | Rounded points for the reference portion                  |
| `pointsPrecise` | float           | Exact `maxpointsSimple` points for the reference portion  |
| `nutrition`  | map<string, float> | Nutritional values per reference portion — see below      |

> **Historical points fields on `defaultPortion`:** `pointsPlusCountPrecise`,
> `smartPointsCountPrecise`, `freestyleCountPrecise`, `tier3CountPrecise`,
> `diabetesCountPrecise`, `maxPoints1CountPrecise`, `maxPoints2CountPrecise`,
> and similar. These reflect how the same food was scored under previous WW
> plans. Safe to ignore unless you need historical comparison.

---

#### `nutrition` map keys

Values are per `defaultPortion.size`. Scale to the tracked serving as described above.

| Key                | Unit    | Meaning                                          |
|--------------------|---------|--------------------------------------------------|
| `calories`         | kcal    | Energy                                           |
| `fat`              | g       | Total fat                                        |
| `saturatedFat`     | g       | Saturated fat                                    |
| `sodium`           | mg      | Sodium                                           |
| `carbs`            | g       | Total carbohydrates                              |
| `carbsWithoutFiber`| g       | Net carbohydrates (carbs minus fibre)            |
| `fiber`            | g       | Dietary fibre                                    |
| `sugar`            | g       | Total sugars                                     |
| `addedSugar`       | g       | Added sugars                                     |
| `sugarAlcohol`     | g       | Sugar alcohols (e.g. erythritol, xylitol)       |
| `erythritol`       | g       | Erythritol specifically                          |
| `protein`          | g       | Protein                                          |
| `alcohol`          | g       | Alcohol                                          |
| `salt`             | g       | Salt (sodium chloride equivalent)                |
| `isEstimatedSugar` | boolean | Whether sugar value is estimated rather than measured |

Not all keys are present for every food — treat missing keys as `0`.

##### Scaling to the tracked serving

```python
scale    = entry["portionSize"] / entry["defaultPortion"]["size"]
calories = entry["defaultPortion"]["nutrition"]["calories"] * scale
protein  = entry["defaultPortion"]["nutrition"]["protein"] * scale
# ... same for all nutrition keys
```

If `defaultPortion.size` is `0` or absent, skip the entry — no usable nutrition data.

---

#### Source types

The `sourceType` field on a food entry identifies where the food definition came from.

| Value               | Meaning                               |
|---------------------|---------------------------------------|
| `WWFOOD`            | WW curated food database              |
| `MEMBERFOOD`        | User-created custom food              |
| `WWVENDORFOOD`      | Branded / vendor-supplied food        |
| `MEMBERRECIPE`      | User-created recipe                   |
| `WWRECIPE`          | WW curated recipe                     |
| `MEMBERFOODQUICK`   | Quick-add (minimal nutritional detail)|

---

## Endpoint status summary

| Status | Method | URL | Purpose |
|--------|--------|-----|---------|
| ✅ Working | `POST` | `auth.weightwatchers.{tld}/login-apis/v1/authenticate` | Credentials → session tokenId |
| ✅ Working | `GET`  | `auth.weightwatchers.{tld}/openam/oauth2/authorize` | Session tokenId → JWT |
| ✅ Working | `GET`  | `cmx.weightwatchers.{tld}/api/v1/cmx/members/~/profile` | Member profile (v1) |
| ✅ Working | `GET`  | `cmx.weightwatchers.{tld}/api/v2/cmx/members/~/profile` | Member profile (v2, identical response) |
| ✅ Working | `GET`  | `cmx.weightwatchers.{tld}/api/v3/cmx/operations/composed/members/~/my-day/{date}` | Daily log, nutrition, points |
| ⚠️ Broken  | `GET`  | `cmx.weightwatchers.{tld}/api/v2/cmx/members/~/activities` | Activity log — backend microservice unreachable (`ENOTFOUND core-activity-service-aa-prod`) |
| ❓ Unknown | `GET`  | `cmx.weightwatchers.{tld}/api/v3/cmx/operations/composed/members/~/my-day/range` | Date-range log — endpoint exists (returns 400), but required query parameter format is unresolved |

**Not found:** Weight history, food search, and per-food nutrition lookup have no
discoverable endpoints under `cmx.weightwatchers.{tld}`. Weight per day is
available via `pointsDetails.weight` in the `my-day` response.
