<?php

namespace App\Http\Controllers;

use App\Models\User;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Http;

class ExportController extends Controller
{
    public function exportData()
    {
        
        // Build the Eloquent query using User Model
        $query = new User;

        // Convert Eloquent query to raw SQL
        $rawSql = vsprintf(str_replace("?", "'%s'", $query->toSql()), $query->getBindings());

        $response = Http::post('http://localhost:8080/export', [
            'query' => $rawSql,
        ]);

        return response()->json($response->json());
    }
}
