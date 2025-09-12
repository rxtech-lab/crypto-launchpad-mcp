"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Plus, X } from "lucide-react";

interface CreateTokenFormProps {
  onSubmit: (tokenData: {
    tokenName: string;
    aud: string[];
    clientId: string;
    roles: string[];
    scopes: string[];
    expiresIn: string;
  }) => Promise<void>;
  isLoading?: boolean;
}

export function CreateTokenForm({ onSubmit, isLoading }: CreateTokenFormProps) {
  const [formData, setFormData] = useState({
    tokenName: "",
    clientId: "",
    expiresIn: "30d",
  });

  const [audiences, setAudiences] = useState<string[]>([]);
  const [roles, setRoles] = useState<string[]>([]);
  const [scopes, setScopes] = useState<string[]>([]);

  const [newAudience, setNewAudience] = useState("");
  const [newRole, setNewRole] = useState("");
  const [newScope, setNewScope] = useState("");

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!formData.tokenName.trim()) {
      alert("Token name is required");
      return;
    }

    await onSubmit({
      tokenName: formData.tokenName,
      aud: audiences,
      clientId: formData.clientId,
      roles,
      scopes,
      expiresIn: formData.expiresIn,
    });

    // Reset form
    setFormData({
      tokenName: "",
      clientId: "",
      expiresIn: "30d",
    });
    setAudiences([]);
    setRoles([]);
    setScopes([]);
  };

  const addItem = (
    value: string,
    setter: React.Dispatch<React.SetStateAction<string[]>>,
    inputSetter: React.Dispatch<React.SetStateAction<string>>
  ) => {
    if (value.trim()) {
      setter((prev) => [...prev, value.trim()]);
      inputSetter("");
    }
  };

  const removeItem = (
    index: number,
    setter: React.Dispatch<React.SetStateAction<string[]>>
  ) => {
    setter((prev) => prev.filter((_, i) => i !== index));
  };

  return (
    <Card className="p-6">
      <h2 className="text-xl font-semibold text-gray-900 mb-4">
        Create New JWT Token
      </h2>

      <form onSubmit={handleSubmit} className="space-y-6">
        {/* Token Name */}
        <div>
          <label
            htmlFor="tokenName"
            className="block text-sm font-medium text-gray-700 mb-2"
          >
            Token Name *
          </label>
          <input
            type="text"
            id="tokenName"
            value={formData.tokenName}
            onChange={(e) =>
              setFormData((prev) => ({ ...prev, tokenName: e.target.value }))
            }
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            placeholder="e.g., API Access Token"
            required
          />
        </div>

        {/* Client ID */}
        <div>
          <label
            htmlFor="clientId"
            className="block text-sm font-medium text-gray-700 mb-2"
          >
            Client ID
          </label>
          <input
            type="text"
            id="clientId"
            value={formData.clientId}
            onChange={(e) =>
              setFormData((prev) => ({ ...prev, clientId: e.target.value }))
            }
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            placeholder="e.g., my-app-client"
          />
        </div>

        {/* Expiration */}
        <div>
          <label
            htmlFor="expiresIn"
            className="block text-sm font-medium text-gray-700 mb-2"
          >
            Expires In
          </label>
          <select
            id="expiresIn"
            value={formData.expiresIn}
            onChange={(e) =>
              setFormData((prev) => ({ ...prev, expiresIn: e.target.value }))
            }
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          >
            <option value="7d">7 days</option>
            <option value="30d">30 days</option>
            <option value="90d">90 days</option>
            <option value="1y">1 year</option>
          </select>
        </div>

        {/* Audiences */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            Audiences
          </label>
          <div className="flex gap-2 mb-2">
            <input
              type="text"
              value={newAudience}
              onChange={(e) => setNewAudience(e.target.value)}
              className="flex-1 px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              placeholder="e.g., api.example.com"
              onKeyPress={(e) => {
                if (e.key === "Enter") {
                  e.preventDefault();
                  addItem(newAudience, setAudiences, setNewAudience);
                }
              }}
            />
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => addItem(newAudience, setAudiences, setNewAudience)}
            >
              <Plus className="h-4 w-4" />
            </Button>
          </div>
          <div className="flex flex-wrap gap-2">
            {audiences.map((aud, index) => (
              <Badge
                key={index}
                variant="secondary"
                className="flex items-center gap-1"
              >
                {aud}
                <button
                  type="button"
                  onClick={() => removeItem(index, setAudiences)}
                  className="ml-1 hover:text-red-600"
                >
                  <X className="h-3 w-3" />
                </button>
              </Badge>
            ))}
          </div>
        </div>

        {/* Roles */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            Roles
          </label>
          <div className="flex gap-2 mb-2">
            <input
              type="text"
              value={newRole}
              onChange={(e) => setNewRole(e.target.value)}
              className="flex-1 px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              placeholder="e.g., admin, user"
              onKeyPress={(e) => {
                if (e.key === "Enter") {
                  e.preventDefault();
                  addItem(newRole, setRoles, setNewRole);
                }
              }}
            />
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => addItem(newRole, setRoles, setNewRole)}
            >
              <Plus className="h-4 w-4" />
            </Button>
          </div>
          <div className="flex flex-wrap gap-2">
            {roles.map((role, index) => (
              <Badge
                key={index}
                variant="secondary"
                className="flex items-center gap-1"
              >
                {role}
                <button
                  type="button"
                  onClick={() => removeItem(index, setRoles)}
                  className="ml-1 hover:text-red-600"
                >
                  <X className="h-3 w-3" />
                </button>
              </Badge>
            ))}
          </div>
        </div>

        {/* Scopes */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            Scopes
          </label>
          <div className="flex gap-2 mb-2">
            <input
              type="text"
              value={newScope}
              onChange={(e) => setNewScope(e.target.value)}
              className="flex-1 px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              placeholder="e.g., read, write, delete"
              onKeyPress={(e) => {
                if (e.key === "Enter") {
                  e.preventDefault();
                  addItem(newScope, setScopes, setNewScope);
                }
              }}
            />
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => addItem(newScope, setScopes, setNewScope)}
            >
              <Plus className="h-4 w-4" />
            </Button>
          </div>
          <div className="flex flex-wrap gap-2">
            {scopes.map((scope, index) => (
              <Badge
                key={index}
                variant="secondary"
                className="flex items-center gap-1"
              >
                {scope}
                <button
                  type="button"
                  onClick={() => removeItem(index, setScopes)}
                  className="ml-1 hover:text-red-600"
                >
                  <X className="h-3 w-3" />
                </button>
              </Badge>
            ))}
          </div>
        </div>

        {/* Submit Button */}
        <div className="flex justify-end">
          <Button type="submit" disabled={isLoading}>
            {isLoading ? "Creating..." : "Create Token"}
          </Button>
        </div>
      </form>
    </Card>
  );
}
