// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

export interface FigmaLinkInfo {
    url: string
    fileKey: string
    nodeId: string
    hash: string
}

export class FigmaUtils {
    // Regex to match Figma design links with node-id parameter
    // Example: https://www.figma.com/design/2MK7mrQPB8CSsdbGO4ifRY/2026-Files-for-IT-Team?node-id=261-10355
    private static readonly FIGMA_LINK_REGEX = /https:\/\/www\.figma\.com\/design\/([a-zA-Z0-9]+)\/[^?]*\?[^#]*node-id=([0-9-]+)/g

    /**
     * Detects all Figma links in the given text
     * @param text The text to search for Figma links
     * @returns Array of FigmaLinkInfo objects
     */
    static detectFigmaLinks(text: string): FigmaLinkInfo[] {
        const links: FigmaLinkInfo[] = []
        const regex = new RegExp(this.FIGMA_LINK_REGEX)
        let match

        while ((match = regex.exec(text)) !== null) {
            const url = match[0]
            const fileKey = match[1]
            const nodeId = match[2]
            const hash = this.generateHash(url)

            links.push({
                url,
                fileKey,
                nodeId,
                hash,
            })
        }

        return links
    }

    /**
     * Generates a hash from a Figma link for file naming
     * @param url The Figma URL to hash
     * @returns A hash string suitable for file naming
     */
    static generateHash(url: string): string {
        // Simple hash function for generating a unique identifier
        let hash = 0
        for (let i = 0; i < url.length; i++) {
            const char = url.charCodeAt(i)
            hash = ((hash << 5) - hash) + char
            hash = hash & hash // Convert to 32bit integer
        }
        return `figma-${Math.abs(hash).toString(36)}`
    }

    /**
     * Converts Figma node-id format (e.g., "261-10355") to API format (e.g., "261:10355")
     * @param nodeId The node ID in URL format
     * @returns The node ID in API format
     */
    static nodeIdToApiFormat(nodeId: string): string {
        return nodeId.replace(/-/g, ':')
    }

    /**
     * Checks if a given URL is a valid Figma design link with node-id
     * @param url The URL to check
     * @returns True if the URL is a valid Figma link
     */
    static isValidFigmaLink(url: string): boolean {
        const regex = new RegExp(this.FIGMA_LINK_REGEX)
        return regex.test(url)
    }

    /**
     * Extracts file key and node ID from a Figma URL
     * @param url The Figma URL
     * @returns Object with fileKey and nodeId, or null if invalid
     */
    static parseFigmaUrl(url: string): {fileKey: string; nodeId: string} | null {
        const regex = new RegExp(this.FIGMA_LINK_REGEX)
        const match = regex.exec(url)

        if (!match) {
            return null
        }

        return {
            fileKey: match[1],
            nodeId: match[2],
        }
    }

    /**
     * Generates a filename for a Figma preview image
     * @param hash The hash of the Figma link
     * @returns Filename for the preview image
     */
    static generatePreviewFilename(hash: string): string {
        return `${hash}.png`
    }
}

