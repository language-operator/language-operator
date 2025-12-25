/*
  Warnings:

  - You are about to drop the column `acceptedAt` on the `OrganizationInvite` table. All the data in the column will be lost.

*/
-- AlterTable
ALTER TABLE "OrganizationInvite" DROP COLUMN "acceptedAt";

-- CreateIndex
CREATE INDEX "Organization_namespace_idx" ON "Organization"("namespace");

-- CreateIndex
CREATE INDEX "User_createdAt_idx" ON "User"("createdAt");
