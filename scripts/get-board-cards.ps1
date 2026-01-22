param(
    [Parameter(Mandatory=$false)]
    [string]$AccessToken = $env:MM_ACCESS_TOKEN,
    
    [Parameter(Mandatory=$false)]
    [string]$ServerUrl = "https://mm.fambear.online",
    
    [Parameter(Mandatory=$false)]
    [string]$BoardId = "bpn1j696qhjg1bfp45x59x57tdr",
    
    [Parameter(Mandatory=$false)]
    [string]$StatusFilter = "In Progress",
    
    [Parameter(Mandatory=$false)]
    [string]$ProjectFilter = "Boards"
)

if (-not $AccessToken) {
    Write-Error "MM_ACCESS_TOKEN is required. Set it as environment variable or pass as parameter."
    exit 1
}

Write-Host "Fetching cards from board $BoardId..." -ForegroundColor Cyan

try {
    $headers = @{
        "Authorization" = "Bearer $AccessToken"
        "X-Requested-With" = "XMLHttpRequest"
    }

    # First, get board metadata to understand property names
    $boardUri = "$ServerUrl/plugins/focalboard/api/v2/boards/$BoardId"
    Write-Host "Fetching board metadata..." -ForegroundColor Gray
    $board = Invoke-RestMethod -Uri $boardUri -Headers $headers -Method Get -ErrorAction Stop

    # Create a mapping of property IDs to names
    $propertyMap = @{}
    $statusPropertyId = $null
    $projectPropertyId = $null
    $projectValueMap = @{}

    foreach ($prop in $board.cardProperties) {
        $propertyMap[$prop.id] = $prop.name
        if ($prop.name -eq "Status") {
            $statusPropertyId = $prop.id
            # Create status value mapping
            $statusValueMap = @{}
            foreach ($option in $prop.options) {
                $statusValueMap[$option.id] = $option.value
            }
        }
        if ($prop.name -like "*Project*" -or $prop.name -eq "Project") {
            $projectPropertyId = $prop.id
            # Create project value mapping
            foreach ($option in $prop.options) {
                $projectValueMap[$option.id] = $option.value
            }
        }
    }

    $uri = "$ServerUrl/plugins/focalboard/api/v2/boards/$BoardId/cards"

    Write-Host "Requesting cards: $uri" -ForegroundColor Gray

    $response = Invoke-RestMethod -Uri $uri -Headers $headers -Method Get -ErrorAction Stop
    
    Write-Host "`nTotal cards fetched: $($response.Count)" -ForegroundColor Green

    $filteredCards = $response | Where-Object {
        $statusMatch = $true
        $projectMatch = $true

        if ($StatusFilter -and $statusPropertyId) {
            $cardStatusId = $_.properties.$statusPropertyId
            if ($cardStatusId -and $statusValueMap.ContainsKey($cardStatusId)) {
                $statusMatch = $statusValueMap[$cardStatusId] -eq $StatusFilter
            } else {
                $statusMatch = $false
            }
        }

        if ($ProjectFilter -and $projectPropertyId) {
            $cardProjectId = $_.properties.$projectPropertyId
            if ($cardProjectId -and $projectValueMap.ContainsKey($cardProjectId)) {
                $projectMatch = $projectValueMap[$cardProjectId] -eq $ProjectFilter
            } else {
                $projectMatch = $false
            }
        }
        
        $statusMatch -and $projectMatch
    }
    
    Write-Host "`nFiltered cards (Status: '$StatusFilter', Project: '$ProjectFilter'): $($filteredCards.Count)" -ForegroundColor Yellow
    
    if ($filteredCards.Count -gt 0) {
        Write-Host "`n=== Cards in '$StatusFilter' status ===" -ForegroundColor Cyan
        
        foreach ($card in $filteredCards) {
            Write-Host "`n---" -ForegroundColor Gray
            Write-Host "Title: $($card.title)" -ForegroundColor White
            Write-Host "ID: $($card.id)" -ForegroundColor Gray

            $createdDate = [DateTimeOffset]::FromUnixTimeMilliseconds($card.createAt).LocalDateTime
            $updatedDate = [DateTimeOffset]::FromUnixTimeMilliseconds($card.updateAt).LocalDateTime
            Write-Host "Created: $createdDate" -ForegroundColor Gray
            Write-Host "Updated: $updatedDate" -ForegroundColor Gray

            if ($card.properties) {
                Write-Host "Properties:" -ForegroundColor Yellow
                $card.properties.PSObject.Properties | ForEach-Object {
                    $propName = if ($propertyMap.ContainsKey($_.Name)) { $propertyMap[$_.Name] } else { $_.Name }
                    $propValue = $_.Value

                    # Try to resolve value if it's a status
                    if ($statusValueMap.ContainsKey($propValue)) {
                        $propValue = $statusValueMap[$propValue]
                    }

                    # Try to resolve value if it's a project
                    if ($projectValueMap.ContainsKey($propValue)) {
                        $propValue = $projectValueMap[$propValue]
                    }

                    Write-Host "  ${propName}: $propValue" -ForegroundColor Gray
                }
            }
        }
    } else {
        Write-Host "`nNo cards found matching the filters." -ForegroundColor Yellow
    }
    
    return $filteredCards
    
} catch {
    Write-Error "Failed to fetch cards: $_"
    Write-Error $_.Exception.Message
    if ($_.Exception.Response) {
        Write-Error "Status Code: $($_.Exception.Response.StatusCode.value__)"
    }
    exit 1
}

